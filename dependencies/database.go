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

	/*
		// Deploy the postgres cluster with helm chart
		postgresCluster, err := helm.NewChart(ctx, "zalando-postgres-cluster", helm.ChartArgs{
			Chart: pulumi.String("zalando-postgres-cluster"),
			Path:  pulumi.String("./dependencies/charts/"),
			//FetchArgs: helm.FetchArgs{
			//	Repo: pulumi.String("https://charts.bitnami.com/bitnami"),
			//},
			Namespace: pulumi.String("postgres"),
			Values: pulumi.Map{
				"global": pulumi.Map{
					"imageRegistry": pulumi.String("docker.io"),
				},
				"image": pulumi.Map{
					"registry": pulumi.String("docker.io"),
					"tag":      pulumi.String("11.12.0-debian-10-r0"),
				},
				"persistentVolume": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"initdbScripts": pulumi.Map{
					"initdb.sql": pulumi.String("CREATE DATABASE dimo;"),
				},
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"metricsExporter": pulumi.Map{
					"image": pulumi.Map{
						"registry": pulumi.String("docker.io"),
						"tag":      pulumi.String("v0.5.0-debian-10-r0"),
					},
				},
				"replication": pulumi.Map{
					"enabled": pulumi.Bool(true),
				},
				"replica": pulumi.Map{
					"replicaCount": pulumi.Int(2),
				},
				"primary": pulumi.Map{
					"replicaCount": pulumi.Int(1),
				},
			},
		}, pulumi.Provider(infrastructure.KubeProvider))
		if err != nil {
			return err
		}

		ctx.Export("postgresCluster", postgresCluster.URN())
	*/

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
