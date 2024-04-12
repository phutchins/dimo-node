package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// Work with the values-prod.yaml file to get the correct values for the environment (for now)
// The configmap helm stuff globs the files in the directory and creates a configmap from them

func InstallContractEventProcessor(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	conf := config.New(ctx, "")
	environmentName := conf.Require("environment")

	//Deploy the users-api from helm chart
	usersApi, err := helm.NewChart(ctx, "contract-event-processor", helm.ChartArgs{
		Chart:     pulumi.String("contract-event-processor"),
		Path:      pulumi.String("./applications/contract-event-processor/charts"),
		Namespace: pulumi.String("contract-event-processor"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"), // We need to push public versions of the images to docker.io
			},
			"image": pulumi.Map{
				"registry":   pulumi.String("docker.io"),
				"tag":        pulumi.String("latest"),
				"pullPolicy": pulumi.String("IfNotPresent"),
				"repository": pulumi.String("dimozone/contract-event-processor"), // build and push from local for now
			},
			"ingress": pulumi.Map{
				"enabled": pulumi.Bool(false),
			},
			"env": pulumi.Map{
				"ENVIRONMENT":         pulumi.String("prod"),
				"KAFKA_BROKERS":       pulumi.String("kafka-" + environmentName + "-dimo-kafka-kafka-brokers:9092"),
				"BLOCK_CONFIRMATIONS": pulumi.Int(5),
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("ContractEventProcessor", usersApi.URN())

	return nil
}
