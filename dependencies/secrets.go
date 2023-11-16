package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallSecretsDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	err = InstallExternalSecrets(ctx, kubeProvider)
	if err != nil {
		return err
	}

	// TODO: Create link to external secret stores (add param to function for secret store location)

	return nil
}

func InstallExternalSecrets(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) error {
	// Install external-secrets helm chart
	externalSecrets, err := helm.NewChart(ctx, "external-secrets", helm.ChartArgs{
		Chart: pulumi.String("external-secrets"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://external-secrets.github.io/kubernetes-external-secrets/"),
		},
		Namespace: pulumi.String("dimo"),
		Values: pulumi.Map{
			"image": pulumi.Map{
				"registry": pulumi.String("docker.io"),
			},
			"imagePullSecrets": pulumi.Array{
				pulumi.String("regcred"),
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("externalSecrets", externalSecrets.URN())

	return nil
}
