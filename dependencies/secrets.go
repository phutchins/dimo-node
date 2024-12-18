package dependencies

import (
	"fmt"

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
	clusterName := conf.Require("cluster-name")
	clusterLocation := conf.Require("location")

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
								"clusterLocation": pulumi.String(clusterLocation),
								"clusterName":     pulumi.String(clusterName),
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

	// Create External Secrets for passwords
	err = ManagePasswordSecrets(ctx, kubeProvider, clusterSecretStore)
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

	// Grant Secret Manager access at the project level using IAM binding instead of member
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

	// Grant Secret Manager viewer role
	_, err = projects.NewIAMBinding(ctx, "secret-viewer-binding", &projects.IAMBindingArgs{
		Project: pulumi.String(projectID),
		Role:    pulumi.String("roles/secretmanager.viewer"),
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
	// Create a Kubernetes Service Account
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

	// Create a Kubernetes ClusterRoleBinding
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
