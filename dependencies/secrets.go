package dependencies

import (
	"fmt"

	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Define global variable for SecretsProvider
var SecretsProvider *helm.Chart

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

// Could set up the GCP or other cloud provider secret store here
func InstallExternalSecrets(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error, SecretsProvider *helm.Chart) {
	// Get project ID from config
	conf := config.New(ctx, "")
	projectID := conf.Require("gcp-project")

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

	// TODO: Move this to configuration
	//serviceAccountName := "dimo-secret-service-account@dimo-dev-401815.iam.gserviceaccount.com"
	serviceAccountName := fmt.Sprintf("dimo-secret-service-account@%s.iam.gserviceaccount.com", projectID)

	// IAM Policy Binding to allow the Kubernetes Service Account to access Secret Manager secrets
	sa, err := serviceaccount.NewAccount(ctx, "secretServiceAccountIam", &serviceaccount.AccountArgs{
		AccountId: pulumi.String("dimo-secret-service-account"),
	})
	if err != nil {
		return err, nil
	}

	_, err = serviceaccount.NewIAMBinding(ctx, "secretServiceAccountIamBinding", &serviceaccount.IAMBindingArgs{
		ServiceAccountId: sa.Name,
		//Role:             pulumi.String("roles/secretmanager.secretAccessor"),
		Role: pulumi.String("roles/iam.serviceAccountUser"),
		Members: pulumi.StringArray{
			pulumi.String("serviceAccount:" + serviceAccountName),
		},
	}, pulumi.DependsOn([]pulumi.Resource{sa}))
	if err != nil {
		return err, nil
	}

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
