package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallDexAuthN(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	//conf := config.New(ctx, "")
	//environmentName := conf.Require("environment")
	//Deploy the users-api from helm chart
	dexAuthN, err := helm.NewRelease(ctx, "dex-auth-n", &helm.ReleaseArgs{
		Chart: pulumi.String("./applications/cluster-helm-charts/charts/dimo-dex"),
		ValueYamlFiles: pulumi.AssetOrArchiveArray{
			pulumi.NewFileAsset("./applications/cluster-helm-charts/charts/dimo-dex/values-prod.yaml"),
		},
		Namespace: pulumi.String("dex"),
		Values: pulumi.Map{
			"ingress": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"hosts": pulumi.Array{
					pulumi.Map{
						"host": pulumi.String("dex-auth-n.dimo.zone"), // TODO: Get host from cloud provider
						// Find how to only pass in identity-api and get the service from the environment
						"paths": pulumi.Array{
							pulumi.Map{
								"path":     pulumi.String("/"),
								"pathType": pulumi.String("ImplementationSpecific"),
							},
						},
					},
				},
			},
			"env": pulumi.Map{
				"BASE_IMAGE_URL": pulumi.String("https://dex-auth-n.dimo.zone/v1"),
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	// usersApi, err := helm.NewChart(ctx, "dex-auth-n", helm.ChartArgs{
	// 	Chart:     pulumi.String("dimo-dex"),
	// 	Path:      pulumi.String("./applications/cluster-helm-charts/charts"),
	// 	Namespace: pulumi.String("dex"),
	// 	ValueYamlFiles: pulumi.String("./applications/cluster-helm-charts/charts/dimo-dex/values-prod.yaml"),
	// 	Values: pulumi.Map{
	// 		"global": pulumi.Map{
	// 			"imageRegistry": pulumi.String("docker.io"), // We need to push public versions of the images to docker.io
	// 		},
	// 		"image": pulumi.Map{
	// 			"registry":   pulumi.String("docker.io"),
	// 			"tag":        pulumi.String("latest"),
	// 			"pullPolicy": pulumi.String("IfNotPResent"),
	// 			"repository": pulumi.String("dimo-network/dimo-dex"), // build and push from local for now
	// 		},
	// 		"ingress": pulumi.Map{
	// 			"enabled": pulumi.Bool(true),
	// 			"hosts": pulumi.Array{
	// 				pulumi.Map{
	// 					"host": pulumi.String("dex-auth-n.dimo.zone"), // TODO: Get host from cloud provider
	// 					// Find how to only pass in identity-api and get the service from the environment
	// 					"paths": pulumi.Array{
	// 						pulumi.Map{
	// 							"path":     pulumi.String("/"),
	// 							"pathType": pulumi.String("ImplementationSpecific"),
	// 						},
	// 					},
	// 				},
	// 			},
	// 		},
	// 		// TODO: Would be nice to have the option to fork from some chain for testing etc... (tenderly)
	// 		// Could use mumbai testnet contracts also in the short term
	// 		// Define sets of addresses at the top level for different environments
	// 		"env": pulumi.Map{
	// 			"BASE_IMAGE_URL": pulumi.String("https://dex-auth-n.dimo.zone/v1"),
	// 		},
	// 	},
	// }, pulumi.Provider(kubeProvider))
	// if err != nil {
	// 	return err
	// }

	ctx.Export("DexAuthN", dexAuthN.URN())

	return nil
}
