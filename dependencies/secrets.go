package dependencies

import (
	"fmt"

	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Define global variable for SecretsProvider
var SecretsProvider *helm.Chart
var ServiceAccountName = "dimo-secret-svc-account"

func InstallSecretsDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (*helm.Chart, error) {
	// Create external-secrets namespace first and wait for it to be ready
	ns, err := corev1.NewNamespace(ctx, "external-secrets", &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("external-secrets"),
			Labels: pulumi.StringMap{
				"name": pulumi.String("external-secrets"),
			},
		},
	}, pulumi.Provider(kubeProvider))

	if err != nil {
		return nil, err
	}

	// Get project ID from config
	conf := config.New(ctx, "")
	projectID := conf.Require("gcp-project")

	// Create GSA and KSA before installing external-secrets
	gsa, err := CreateGSA(ctx, kubeProvider, projectID)
	if err != nil {
		return nil, err
	}

	ksa, err := CreateKSA(ctx, kubeProvider, gsa.Email, ns)
	if err != nil {
		return nil, err
	}

	// Install external-secrets helm chart with explicit namespace dependency
	SecretsProvider, err = helm.NewChart(ctx, "external-secrets", helm.ChartArgs{
		Chart: pulumi.String("external-secrets"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://charts.external-secrets.io/"),
		},
		Namespace: pulumi.String("external-secrets"),
		Values: pulumi.Map{
			"installCRDs": pulumi.Bool(true),
			"webhook": pulumi.Map{
				"create": pulumi.Bool(true),
				"port":   pulumi.Int(9443),
				"service": pulumi.Map{
					"type": pulumi.String("ClusterIP"),
					"ports": pulumi.Map{
						"port":       pulumi.Int(443),
						"targetPort": pulumi.Int(9443),
					},
				},
			},
			"serviceAccount": pulumi.Map{
				"create": pulumi.Bool(false),
				"name":   pulumi.String("external-secrets-ksa"),
			},
			"podLabels": pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("external-secrets"),
			},
			"priorityClassName": pulumi.String("gmp-critical"),
		},
	}, pulumi.Provider(kubeProvider),
		pulumi.DependsOn([]pulumi.Resource{ns, ksa}))

	if err != nil {
		return nil, err
	}

	// Wait a bit for the webhook to be ready
	// webhookWait, err := corev1.NewConfigMap(ctx, "webhook-wait", &corev1.ConfigMapArgs{
	// 	Metadata: &metav1.ObjectMetaArgs{
	// 		Name:      pulumi.String("webhook-wait"),
	// 		Namespace: pulumi.String("external-secrets"),
	// 	},
	// 	Data: pulumi.StringMap{
	// 		"wait": pulumi.String("true"),
	// 	},
	// }, pulumi.Provider(kubeProvider),
	// 	pulumi.DependsOn([]pulumi.Resource{SecretsProvider}))

	// if err != nil {
	// 	return err, nil
	// }

	// Create ClusterSecretStore after webhook is ready
	clusterSecretStore, err := apiextensions.NewCustomResource(ctx, "ClusterSecretStore", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("external-secrets.io/v1beta1"),
		Kind:       pulumi.String("ClusterSecretStore"),
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("cluster-secret-store"),
		},
		OtherFields: map[string]interface{}{
			"spec": map[string]interface{}{
				"provider": map[string]interface{}{
					"gcpsm": map[string]interface{}{
						"projectID": pulumi.String(projectID),
						"auth": map[string]interface{}{
							"workloadIdentity": map[string]interface{}{
								"serviceAccountRef": map[string]interface{}{
									"name":      "external-secrets-ksa",
									"namespace": "external-secrets",
								},
								"clusterLocation": pulumi.String("europe-west1-b"),
								"clusterName":     pulumi.String("dimo-eu-cd89583"),
							},
						},
					},
				},
			},
		},
	}, pulumi.Provider(kubeProvider),
		pulumi.DependsOn([]pulumi.Resource{SecretsProvider, ns}))

	if err != nil {
		return nil, err
	}

	ctx.Export("externalSecrets", SecretsProvider.URN())
	ctx.Export("externalSecret", clusterSecretStore.URN())

	// Create database secrets after ClusterSecretStore is ready
	err = createDatabaseSecrets(ctx, kubeProvider, clusterSecretStore)
	if err != nil {
		return nil, err
	}

	return SecretsProvider, nil
}

// If we're not using roles, use a service account key
func CreateESSecrets(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) error {
	// Create a secret to store the GCP service account key
	_, err := corev1.NewSecret(ctx, "secret-service-account-key", &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("secret-service-account-key"),
			Namespace: pulumi.String("external-secrets"),
		},
		StringData: pulumi.StringMap{
			"KEYID":     pulumi.String(""),
			"SECRETKEY": pulumi.String(""),
		},
		Type: pulumi.String("Opaque"),
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	return nil
}

func CreateGSA(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, projectID string) (gsa *serviceaccount.Account, err error) {
	// Create the service account
	gsa, err = serviceaccount.NewAccount(ctx, "secret-account", &serviceaccount.AccountArgs{
		AccountId: pulumi.String(ServiceAccountName),
	})
	if err != nil {
		return nil, err
	}

	// Grant workload identity user role to the KSA
	_, err = serviceaccount.NewIAMMember(ctx, "workload-identity-binding", &serviceaccount.IAMMemberArgs{
		ServiceAccountId: gsa.Email.ApplyT(func(email string) string {
			return fmt.Sprintf("projects/%s/serviceAccounts/%s", projectID, email)
		}).(pulumi.StringOutput),
		Role:   pulumi.String("roles/iam.workloadIdentityUser"),
		Member: pulumi.String(fmt.Sprintf("serviceAccount:%s.svc.id.goog[external-secrets/external-secrets-ksa]", projectID)),
	})
	if err != nil {
		return nil, err
	}

	// Commenting out until I get the permissions needed in GCP
	// // Grant Secret Manager access at the project level using IAM binding instead of member
	_, err = projects.NewIAMBinding(ctx, "secret-accessor-binding", &projects.IAMBindingArgs{
		Project: pulumi.String(projectID),
		Role:    pulumi.String("roles/secretmanager.secretAccessor"),
		Members: pulumi.StringArray{
			gsa.Email.ApplyT(func(email string) string {
				return fmt.Sprintf("serviceAccount:%s", email)
			}).(pulumi.StringOutput),
		},
	})
	if err != nil {
		return nil, err
	}

	return gsa, nil
}

func CreateKSA(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, gsaEmail pulumi.StringOutput, ns *corev1.Namespace) (ksa *corev1.ServiceAccount, err error) {
	// ksa, err = corev1.NewServiceAccount(ctx, "external-secrets", &corev1.ServiceAccountArgs{
	// 	Metadata: &metav1.ObjectMetaArgs{
	// 		Name:      pulumi.String("external-secrets"),
	// 		Namespace: pulumi.String("external-secrets"),
	// 		Annotations: pulumi.StringMap{
	// 			"iam.gke.io/gcp-service-account": gsaEmail,
	// 		},
	// 	},
	// })

	// 3. Create a Kubernetes Service Account
	ksa, err = corev1.NewServiceAccount(ctx, "secret-service-account", &corev1.ServiceAccountArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Namespace: pulumi.String("external-secrets"),
			Name:      pulumi.String("external-secrets-ksa"),
			Annotations: pulumi.StringMap{
				"iam.gke.io/gcp-service-account": gsaEmail,
			},
		},
	}, pulumi.Provider(kubeProvider),
		pulumi.DependsOn([]pulumi.Resource{ns}))
	if err != nil {
		return nil, err
	}

	// 4. Create a Kubernetes ClusterRoleBinding
	_, err = rbacv1.NewClusterRoleBinding(ctx, "secret-service-account-binding", &rbacv1.ClusterRoleBindingArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("secret-service-account-binding"),
		},
		RoleRef: &rbacv1.RoleRefArgs{
			ApiGroup: pulumi.String("rbac.authorization.k8s.io"),
			Kind:     pulumi.String("ClusterRole"),
			Name:     pulumi.String("cluster-admin"), // Adjust as needed for appropriate permissions
		},
		Subjects: rbacv1.SubjectArray{
			&rbacv1.SubjectArgs{
				Kind:      pulumi.String("ServiceAccount"),
				Name:      ksa.Metadata.Name().Elem(),
				Namespace: pulumi.String("external-secrets"),
			},
		},
	}, pulumi.Provider(kubeProvider),
		pulumi.DependsOn([]pulumi.Resource{ns}))
	if err != nil {
		return nil, err
	}

	return ksa, nil
}

// Could set up the GCP or other cloud provider secret store here
func InstallExternalSecrets(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, ns *corev1.Namespace) (*helm.Chart, error) {
	// Get project ID from config
	conf := config.New(ctx, "")
	projectID := conf.Require("gcp-project")

	gsa, err := CreateGSA(ctx, kubeProvider, projectID)
	if err != nil {
		return nil, err
	}

	ksa, err := CreateKSA(ctx, kubeProvider, gsa.Email, ns)
	if err != nil {
		return nil, err
	}

	ctx.Export("ksa", ksa.URN())

	// Install external-secrets helm chart
	SecretsProvider, err = helm.NewChart(ctx, "external-secrets", helm.ChartArgs{
		Chart: pulumi.String("external-secrets"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://charts.external-secrets.io/"),
		},
		Namespace: pulumi.String("external-secrets"),
		Values: pulumi.Map{
			"installCRDs": pulumi.Bool(true),
			// "webhook": pulumi.Map{
			// 	"create": pulumi.Bool(true),
			// 	"port":   pulumi.Int(9443),
			// 	"service": pulumi.Map{
			// 		"name": pulumi.String("external-secrets-webhook"),
			// 		"type": pulumi.String("ClusterIP"),
			// 	},
			// },
			"serviceAccount": pulumi.Map{
				"create": pulumi.Bool(false),
				"name":   pulumi.String("external-secrets-ksa"),
			},
			"priorityClassName": pulumi.String("gmp-critical"),
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return nil, err
	}

	ctx.Export("externalSecrets", SecretsProvider.URN())

	return SecretsProvider, nil
}

func createDatabaseSecrets(ctx *pulumi.Context, provider *kubernetes.Provider, clusterSecretStore *apiextensions.CustomResource) error {
	stack := ctx.Stack()

	// Get passwords from Pulumi config
	_, err := utils.GetPassword(stack, "postgres-root")
	if err != nil {
		return fmt.Errorf("failed to get postgres root password: %v", err)
	}

	_, err = utils.GetPassword(stack, "identity-api-db")
	if err != nil {
		return fmt.Errorf("failed to get identity-api password: %v", err)
	}

	// Create ExternalSecret for PostgreSQL root password
	_, err = apiextensions.NewCustomResource(ctx, "postgres-root-secret", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("external-secrets.io/v1beta1"),
		Kind:       pulumi.String("ExternalSecret"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("postgres-root-secret"),
			Namespace: pulumi.String("default"),
		},
		OtherFields: map[string]interface{}{
			"spec": map[string]interface{}{
				"refreshInterval": "1h",
				"secretStoreRef": map[string]interface{}{
					"name": "cluster-secret-store",
					"kind": "ClusterSecretStore",
				},
				"target": map[string]interface{}{
					"name": "postgres-root-secret",
				},
				"data": []map[string]interface{}{
					{
						"secretKey": "password",
						"remoteRef": map[string]interface{}{
							"key":      fmt.Sprintf("projects/%s/secrets/postgres-root-password", ctx.Stack()),
							"property": "latest",
						},
					},
				},
			},
		},
	}, pulumi.Provider(provider),
		pulumi.DependsOn([]pulumi.Resource{clusterSecretStore}))
	if err != nil {
		return fmt.Errorf("failed to create postgres root secret: %v", err)
	}

	// Create ExternalSecret for Identity API database password
	_, err = apiextensions.NewCustomResource(ctx, "identity-api-db-secret", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("external-secrets.io/v1beta1"),
		Kind:       pulumi.String("ExternalSecret"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("identity-api-db-secret"),
			Namespace: pulumi.String("default"),
		},
		OtherFields: map[string]interface{}{
			"spec": map[string]interface{}{
				"refreshInterval": "1h",
				"secretStoreRef": map[string]interface{}{
					"name": "cluster-secret-store",
					"kind": "ClusterSecretStore",
				},
				"target": map[string]interface{}{
					"name": "identity-api-db-secret",
				},
				"data": []map[string]interface{}{
					{
						"secretKey": "password",
						"remoteRef": map[string]interface{}{
							"key":      fmt.Sprintf("projects/%s/secrets/identity-api-db-password", ctx.Stack()),
							"property": "latest",
						},
					},
				},
			},
		},
	}, pulumi.Provider(provider),
		pulumi.DependsOn([]pulumi.Resource{clusterSecretStore}))
	if err != nil {
		return fmt.Errorf("failed to create identity-api db secret: %v", err)
	}

	return nil
}
