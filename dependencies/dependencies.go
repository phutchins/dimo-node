package dependencies

import (
	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallDependencies(ctx *pulumi.Context, provider *kubernetes.Provider) (error, *helm.Chart) {
	// Install nginx-ingress
	if err := InstallNginxIngress(ctx, provider); err != nil {
		return err, nil
	}

	// Install cert-manager and configure Let's Encrypt
	if err := InstallLetsEncrypt(ctx, provider); err != nil {
		return err, nil
	}

	// Install external-secrets operator and get the ClusterSecretStore
	secretsProvider, err := InstallSecretsDependencies(ctx, provider)
	if err != nil {
		return err, nil
	}

	return nil, secretsProvider
}

func InstallNginxIngress(ctx *pulumi.Context, provider *kubernetes.Provider) error {
	// Create namespace for nginx-ingress
	namespaces, err := utils.CreateNamespaces(ctx, provider, []string{"ingress-nginx"})
	if err != nil {
		return err
	}

	// Install main nginx-ingress controller
	_, err = helm.NewChart(ctx, "ingress-nginx", helm.ChartArgs{
		Chart: pulumi.String("ingress-nginx"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://kubernetes.github.io/ingress-nginx"),
		},
		Version:   pulumi.String("4.11.2"),
		Namespace: pulumi.String("ingress-nginx"),
		Values: pulumi.Map{
			"controller": pulumi.Map{
				"image": pulumi.Map{
					"chroot": pulumi.Bool(true),
				},
				"kind":         pulumi.String("Deployment"),
				"replicaCount": pulumi.Int(2),
				"ingressClassResource": pulumi.Map{
					"name":            pulumi.String("nginx"),
					"enabled":         pulumi.Bool(true),
					"default":         pulumi.Bool(false),
					"controllerValue": pulumi.String("k8s.io/ingress-nginx"),
				},
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
				"resources": pulumi.Map{
					"requests": pulumi.Map{
						"cpu":    pulumi.String("100m"),
						"memory": pulumi.String("200Mi"),
					},
					"limits": pulumi.Map{
						"cpu":    pulumi.String("500m"),
						"memory": pulumi.String("500Mi"),
					},
				},
			},
		},
	}, pulumi.Provider(provider), pulumi.DependsOn([]pulumi.Resource{namespaces["ingress-nginx"]}))

	if err != nil {
		return err
	}

	return nil
}
