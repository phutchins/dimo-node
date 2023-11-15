package dependencies

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Define variables needed globally in the dependencies package

func InstallKafka(ctx *pulumi.Context) (err error) {
	/*

		// Install Kafka operator and cluster
		kafkaOperator, err := helm.NewChart(ctx, "kafka-operator", helm.ChartArgs{
			Chart: pulumi.String("kafka-operator"),
			FetchArgs: helm.FetchArgs{
				Repo: pulumi.String("https://charts.bitnami.com/bitnami"),
			},
			Namespace: pulumi.String("dimo"),
			Values: pulumi.Map{
				"global": pulumi.Map{
					"imageRegistry": pulumi.String("docker.io"),
				},
				"operator": pulumi.Map{
					"image": pulumi.Map{
						"repository": pulumi.String("bitnami/kafka-operator"),
						"tag":        pulumi.String("0.3.0-debian-10-r0"),
					},
				},
			},
		}, pulumi.Provider(infrastructure.KubeProvider))
		if err != nil {
			return err
		}

		kafkaCluster, err := helm.NewChart(ctx, "kafka-cluster", helm.ChartArgs{
			Chart: pulumi.String("kafka-cluster"),
			FetchArgs: helm.FetchArgs{
				Repo: pulumi.String("https://charts.bitnami.com/bitnami"),
			},
			Namespace: pulumi.String("dimo"),
			Values: pulumi.Map{
				"global": pulumi.Map{
					"imageRegistry": pulumi.String("docker.io"),
				},
				"image": pulumi.Map{
					"registry": pulumi.String("docker.io"),
					"tag":      pulumi.String("2.8.0-debian-10-r0"),
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

		ctx.Export("kafkaOperator", kafkaOperator.URN())
		ctx.Export("kafkaCluster", kafkaCluster.URN())

	*/

	return nil
}
