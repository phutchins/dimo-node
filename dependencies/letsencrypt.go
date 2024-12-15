package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallCertManager(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (*helm.Release, error) {
	// Install cert-manager helm chart
	certManager, err := helm.NewRelease(ctx, "cert-manager", &helm.ReleaseArgs{
		Chart: pulumi.String("cert-manager"),
		RepositoryOpts: &helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://charts.jetstack.io/"),
		},
		Namespace: pulumi.String("cert-manager"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"),
			},
			"installCRDs": pulumi.Bool(true),
			"webhook": pulumi.Map{
				"timeoutSeconds": pulumi.Int(30),
			},
		},
		SkipAwait:     pulumi.Bool(false), // Wait for resources to be ready
		WaitForJobs:   pulumi.Bool(true),  // Wait for any jobs to complete
		CleanupOnFail: pulumi.Bool(true),
		Replace:       pulumi.Bool(true),
	}, pulumi.Provider(kubeProvider))

	if err != nil {
		return nil, err
	}

	ctx.Export("certManager", certManager.URN())
	return certManager, nil
}

func InstallLetsEncrypt(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) error {
	// Use the cert-manager installation from certificates.go
	_, err := InstallCertManager(ctx, kubeProvider)
	if err != nil {
		return err
	}

	// // Create ClusterIssuer for Let's Encrypt
	// _, err = apiextensions.NewCustomResource(ctx, "letsencrypt-prod", &apiextensions.CustomResourceArgs{
	// 	ApiVersion: pulumi.String("cert-manager.io/v1"),
	// 	Kind:       pulumi.String("ClusterIssuer"),
	// 	Metadata: &metav1.ObjectMetaArgs{
	// 		Name: pulumi.String("letsencrypt-prod"),
	// 	},
	// 	OtherFields: map[string]interface{}{
	// 		"spec": map[string]interface{}{
	// 			"acme": map[string]interface{}{
	// 				"server": "https://acme-v02.api.letsencrypt.org/directory",
	// 				"email":  "admin@dimo.zone",
	// 				"privateKeySecretRef": map[string]interface{}{
	// 					"name": "letsencrypt-prod",
	// 				},
	// 				"solvers": []map[string]interface{}{
	// 					{
	// 						"http01": map[string]interface{}{
	// 							"ingress": map[string]interface{}{
	// 								"class": "nginx",
	// 							},
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 	},
	// }, pulumi.Provider(kubeProvider),
	// 	pulumi.DependsOn([]pulumi.Resource{certManager}))

	// return err
	return nil
}
