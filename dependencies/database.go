package dependencies

import (
	"github.com/dimo/dimo-node/infrastructure"
	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Define variables needed globally in the dependencies package

func InstallDatabaseDependencies(ctx *pulumi.Context) (err error) {
	err = utils.CreateNamespaces(ctx, infrastructure.KubeProvider, []string{"postgres"})
	if err != nil {
		return err
	}

	// Maybe use the crunchyroll operator instead of the zalando one
	// https://www.pulumi.com/ai/answers/s7UarM5px7wLQgmwWRmrMB/orchestrating-postgresql-on-kubernetes-with-crunchydatas-operator
	// https://access.crunchydata.com/documentation/postgres-operator/latest
	// Deploy the postgres operator with helm chart
	postgresOperator, err := helm.NewChart(ctx, "postgres-operator", helm.ChartArgs{
		Chart: pulumi.String("postgres-operator"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://opensource.zalando.com/postgres-operator/charts/postgres-operator"),
		},
		Namespace: pulumi.String("postgres"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"),
			},
			"operator": pulumi.Map{
				"image": pulumi.Map{
					"repository": pulumi.String("bitnami/postgres-operator"),
					"tag":        pulumi.String("1.7.0-debian-10-r0"),
				},
			},
		},
	}, pulumi.Provider(infrastructure.KubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("postgresOperator", postgresOperator.URN())

	// Deploy the postgres operator ui helm chart
	postgresOperatorUI, err := helm.NewChart(ctx, "postgres-operator-ui", helm.ChartArgs{
		Chart: pulumi.String("postgres-operator-ui"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://opensource.zalando.com/postgres-operator/charts/postgres-operator-ui"),
		},
		Namespace: pulumi.String("postgres"),
		Values: pulumi.Map{
			/*
				"global": pulumi.Map{
					"imageRegistry": pulumi.String("docker.io"),
				},
				"image": pulumi.Map{
					"registry": pulumi.String("docker.io"),
					"tag":      pulumi.String("1.7.0-debian-10-r0"),
				}, */
			"service": pulumi.Map{
				"type": pulumi.String("ClusterIP"),
			},
		},
	}, pulumi.Provider(infrastructure.KubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("postgresOperatorUI", postgresOperatorUI.URN())

	// Deploy a postgres cluster with a custom resource definition
	postgresClusterCRD, err := yaml.NewConfigFile(ctx, "postgres-cluster-crd", &yaml.ConfigFileArgs{
		File: string("./dependencies/crds/postgres-cluster-crd.yaml"),
	}, pulumi.Provider(infrastructure.KubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("postgresClusterCRD", postgresClusterCRD.URN())

	return nil
}
