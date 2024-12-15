package dependencies

import (
	"fmt"

	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	secretsmanager "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/external-secrets"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	rbacv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/rbac/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Define global variable for SecretsProvider
var SecretsProvider *helm.Chart
var ServiceAccountName = "dimo-secret-svc-account"

func InstallSecretsDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error, SecretsProvider *helm.Chart) {
	err = utils.CreateNamespaces(ctx, kubeProvider, []string{"external-secrets"})
	if err != nil {
		return err, nil
	}

	//err = CreateESSecrets(ctx, kubeProvider)

	err, SecretsProvider = InstallExternalSecrets(ctx, kubeProvider)
	if err != nil {
		return err, nil
	}

	// TODO: Create link to external secret stores (add param to function for secret store location)

	return nil, SecretsProvider
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

	return gsa, nil
}

func CreateKSA(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, gsaEmail pulumi.StringOutput) (ksa *corev1.ServiceAccount, err error) {
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
	}, pulumi.Provider(kubeProvider))
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
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return nil, err
	}

	return ksa, nil
}

// Could set up the GCP or other cloud provider secret store here
func InstallExternalSecrets(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error, SecretsProvider *helm.Chart) {
	// Get project ID from config
	conf := config.New(ctx, "")
	projectID := conf.Require("gcp-project")
	// TODO: Move this to configuration
	//serviceAccountName := "dimo-secret-svc-account@dimo-dev-401815.iam.gserviceaccount.com"
	//serviceAccountName := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", ServiceAccountName, projectID)

	gsa, err := CreateGSA(ctx, kubeProvider, projectID)
	if err != nil {
		return err, nil
	}

	ksa, err := CreateKSA(ctx, kubeProvider, gsa.Email)
	if err != nil {
		return err, nil
	}

	// Create GSA and grant permissions
	// gsa, err := iam.NewServiceAccount(ctx, "external-secrets-sa", &gcp.ServiceAccountArgs{
	// 	AccountId:   pulumi.String("external-secrets-sa"),
	// 	DisplayName: pulumi.String("External Secrets Service Account"),
	// })

	// _, err = gcp.NewProjectIAMMember(ctx, "secret-accessor", &gcp.ProjectIAMMemberArgs{
	// 	Project: pulumi.String("your-project-id"),
	// 	Role:    pulumi.String("roles/secretmanager.secretAccessor"),
	// 	Member: gsa.Email.ApplyT(func(email string) string {
	// 		return fmt.Sprintf("serviceAccount:%s", email)
	// 	}).(pulumi.StringOutput),
	// })

	// Create KSA with workload identity annotation
	// ksa, err := corev1.NewServiceAccount(ctx, "external-secrets", &corev1.ServiceAccountArgs{
	// 	Metadata: &metav1.ObjectMetaArgs{
	// 		Name:      pulumi.String("external-secrets"),
	// 		Namespace: pulumi.String("external-secrets"),
	// 		Annotations: pulumi.StringMap{
	// 			"iam.gke.io/gcp-service-account": gsa.Email,
	// 		},
	// 	},
	// })

	ctx.Export("ksa", ksa.URN())

	// Bind GSA to KSA
	// _, err = gcp.NewServiceAccountIAMMember(ctx, "workload-identity-binding", &gcp.ServiceAccountIAMMemberArgs{
	// 	ServiceAccountId: gsa.Name,
	// 	Role:             pulumi.String("roles/iam.workloadIdentityUser"),
	// 	Member: pulumi.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]",
	// 		"your-project-id",
	// 		"external-secrets",
	// 		"external-secrets"),
	// })

	// Install external-secrets helm chart
	SecretsProvider, err = helm.NewChart(ctx, "external-secrets", helm.ChartArgs{
		Chart: pulumi.String("external-secrets"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://charts.external-secrets.io/"),
		},
		Namespace: pulumi.String("external-secrets"),
		Values: pulumi.Map{
			"installCRDs": pulumi.Bool(true),
			"crds": pulumi.Map{
				"createClusterSecretStore": pulumi.Bool(true),
			},
			"priorityClassName": pulumi.String("gmp-critical"),
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err, nil
	}

	// Seems that the ClusterSecretStore resource definition is not being created
	// so had to manually create it

	ctx.Export("externalSecrets", SecretsProvider.URN())

	//exampleSecret := pulumi.String("cluster-secret")
	//exampleSecretVersion := pulumi.String("cluster-secret-version")

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
					},
				},
			},
		},
	}, pulumi.Provider(kubeProvider), pulumi.DependsOn([]pulumi.Resource{SecretsProvider}))
	if err != nil {
		return err, nil
	}

	// Add asure key store and AWS here (switch and set up)
	// Consider how to migrate from cluster secret store to namespace store

	ctx.Export("externalSecret", clusterSecretStore.URN())

	// Add this to InstallExternalSecrets before creating any ExternalSecret resources
	_, err = corev1.NewService(ctx, "wait-for-webhook", &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("wait-for-webhook"),
			Namespace: pulumi.String("external-secrets"),
		},
		Spec: &corev1.ServiceSpecArgs{
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(443),
					TargetPort: pulumi.Int(443),
				},
			},
			Selector: pulumi.StringMap{
				"app.kubernetes.io/name": pulumi.String("external-secrets"),
			},
		},
	}, pulumi.Provider(kubeProvider),
		pulumi.DependsOn([]pulumi.Resource{SecretsProvider}))

	return nil, SecretsProvider
}

func createDatabaseSecrets(ctx *pulumi.Context, provider *kubernetes.Provider) error {
	stack := ctx.Stack()

	// Get passwords from Pulumi config
	postgresRootPass, err := utils.GetPassword(stack, "postgres-root")
	if err != nil {
		return fmt.Errorf("failed to get postgres root password: %v", err)
	}

	identityApiPass, err := utils.GetPassword(stack, "identity-api-db")
	if err != nil {
		return fmt.Errorf("failed to get identity-api password: %v", err)
	}

	// Create ExternalSecret for PostgreSQL root password
	_, err = secretsmanager.NewExternalSecret(ctx, "postgres-root-secret", &secretsmanager.ExternalSecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("postgres-root-secret"),
			Namespace: pulumi.String("default"),
		},
		Spec: &secretsmanager.ExternalSecretSpecArgs{
			RefreshInterval: pulumi.String("1h"),
			SecretStoreRef: &secretsmanager.ExternalSecretSpecSecretStoreRefArgs{
				Name: pulumi.String("gcp-store"),
				Kind: pulumi.String("ClusterSecretStore"),
			},
			Target: &secretsmanager.ExternalSecretSpecTargetArgs{
				Name: pulumi.String("postgres-root-secret"),
			},
			Data: secretsmanager.ExternalSecretSpecDataArray{
				&secretsmanager.ExternalSecretSpecDataArgs{
					SecretKey: pulumi.String("password"),
					RemoteRef: &secretsmanager.ExternalSecretSpecDataRemoteRefArgs{
						Key:      pulumi.String(fmt.Sprintf("projects/%s/secrets/postgres-root-password", ctx.Stack())),
						Property: pulumi.String("latest"),
					},
				},
			},
		},
	}, pulumi.Provider(provider))
	if err != nil {
		return fmt.Errorf("failed to create postgres root secret: %v", err)
	}

	// Create ExternalSecret for Identity API database password
	_, err = secretsmanager.NewExternalSecret(ctx, "identity-api-db-secret", &secretsmanager.ExternalSecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("identity-api-db-secret"),
			Namespace: pulumi.String("default"),
		},
		Spec: &secretsmanager.ExternalSecretSpecArgs{
			RefreshInterval: pulumi.String("1h"),
			SecretStoreRef: &secretsmanager.ExternalSecretSpecSecretStoreRefArgs{
				Name: pulumi.String("gcp-store"),
				Kind: pulumi.String("ClusterSecretStore"),
			},
			Target: &secretsmanager.ExternalSecretSpecTargetArgs{
				Name: pulumi.String("identity-api-db-secret"),
			},
			Data: secretsmanager.ExternalSecretSpecDataArray{
				&secretsmanager.ExternalSecretSpecDataArgs{
					SecretKey: pulumi.String("password"),
					RemoteRef: &secretsmanager.ExternalSecretSpecDataRemoteRefArgs{
						Key:      pulumi.String(fmt.Sprintf("projects/%s/secrets/identity-api-db-password", ctx.Stack())),
						Property: pulumi.String("latest"),
					},
				},
			},
		},
	}, pulumi.Provider(provider))
	if err != nil {
		return fmt.Errorf("failed to create identity-api db secret: %v", err)
	}

	return nil
}
