package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallCertManager(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (*helm.Release, error) {
	// Create cert-manager namespace first and wait for it to be ready
	ns, err := corev1.NewNamespace(ctx, "cert-manager", &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("cert-manager"),
		},
	}, pulumi.Provider(kubeProvider))

	if err != nil {
		return nil, err
	}

	// Install cert-manager with CRDs and all components
	certManager, err := helm.NewRelease(ctx, "cert-manager", &helm.ReleaseArgs{
		Chart:   pulumi.String("cert-manager"),
		Version: pulumi.String("v1.13.2"),
		RepositoryOpts: &helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://charts.jetstack.io/"),
		},
		Namespace:       pulumi.String("cert-manager"),
		CreateNamespace: pulumi.Bool(false),
		Values: pulumi.Map{
			"installCRDs": pulumi.Bool(true),
			"webhook": pulumi.Map{
				"timeoutSeconds": pulumi.Int(30),
			},
			"startupapicheck": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"resources": pulumi.Map{
				"requests": pulumi.Map{
					"cpu":    pulumi.String("10m"),
					"memory": pulumi.String("32Mi"),
				},
				"limits": pulumi.Map{
					"cpu":    pulumi.String("100m"),
					"memory": pulumi.String("128Mi"),
				},
			},
		},
		SkipAwait:     pulumi.Bool(false),
		WaitForJobs:   pulumi.Bool(false),
		CleanupOnFail: pulumi.Bool(true),
		Timeout:       pulumi.Int(600),
		Replace:       pulumi.Bool(true),
	}, pulumi.Provider(kubeProvider),
		pulumi.DependsOn([]pulumi.Resource{ns}))

	if err != nil {
		return nil, err
	}

	ctx.Export("certManager", certManager.URN())
	return certManager, nil
}

func InstallLetsEncrypt(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) error {
	// Use the cert-manager installation from certificates.go
	certManager, err := InstallCertManager(ctx, kubeProvider)
	if err != nil {
		return err
	}

	// Create ClusterIssuer for Let's Encrypt
	_, err = apiextensions.NewCustomResource(ctx, "letsencrypt-prod", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("ClusterIssuer"),
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("letsencrypt-prod"),
		},
		OtherFields: map[string]interface{}{
			"spec": map[string]interface{}{
				"acme": map[string]interface{}{
					"server": "https://acme-v02.api.letsencrypt.org/directory",
					"email":  "admin@driveomid.xyz",
					"privateKeySecretRef": map[string]interface{}{
						"name": "letsencrypt-prod",
					},
					"solvers": []map[string]interface{}{
						{
							"http01": map[string]interface{}{
								"ingress": map[string]interface{}{
									"class": "nginx",
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Provider(kubeProvider),
		pulumi.DependsOn([]pulumi.Resource{certManager}))

	return err
}
