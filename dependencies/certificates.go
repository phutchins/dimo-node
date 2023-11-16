package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallCertificateDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	err = InstallCertManager(ctx, kubeProvider)
	if err != nil {
		return err
	}

	err = InstallOriginCAIssuer(ctx, kubeProvider)
	if err != nil {
		return err
	}

	return nil
}

func InstallCertManager(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) error {
	// Install cert-manager helm chart
	certManager, err := helm.NewChart(ctx, "cert-manager", helm.ChartArgs{
		Chart: pulumi.String("cert-manager"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://charts.jetstack.io"),
		},
		Namespace: pulumi.String("dimo"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"),
			},
			"installCRDs": pulumi.Bool(true),
			"imagePullSecrets": pulumi.Array{
				pulumi.String("regcred"),
			},
			"webhook": pulumi.Map{
				"image": pulumi.Map{
					"imagePullPolicy": pulumi.String("IfNotPresent"),
				},
			},
			"certManager": pulumi.Map{
				"image": pulumi.Map{
					"imagePullPolicy": pulumi.String("IfNotPresent"),
				},
			},
			"cainjector": pulumi.Map{
				"image": pulumi.Map{
					"imagePullPolicy": pulumi.String("IfNotPresent"),
				},
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("certManager", certManager.URN())

	return nil
}

func InstallOriginCAIssuer(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) error {
	// Install origin-ca-issuer helm chart
	originCAIssuer, err := helm.NewChart(ctx, "origin-ca-issuer", helm.ChartArgs{
		Chart: pulumi.String("origin-ca-issuer"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://charts.jetstack.io"),
		},
		Namespace: pulumi.String("dimo"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"),
			},
			"imagePullSecrets": pulumi.Array{
				pulumi.String("regcred"),
			},
			"image": pulumi.Map{
				"imagePullPolicy": pulumi.String("IfNotPresent"),
			},
			"ca": pulumi.Map{
				"secretName": pulumi.String("origin-ca"),
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("originCAIssuer", originCAIssuer.URN())

	return nil
}
