package dependencies

import (
	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallSecretsDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	err = utils.CreateNamespaces(ctx, kubeProvider, []string{"external-secrets"})
	if err != nil {
		return err
	}

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
			Repo: pulumi.String("https://charts.external-secrets.io/"),
		},
		Namespace: pulumi.String("external-secrets"),
		Values: pulumi.Map{
			"installCRDs":       pulumi.Bool(true),
			"priorityClassName": pulumi.String("high-priority"),
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("externalSecrets", externalSecrets.URN())

	return nil
}
