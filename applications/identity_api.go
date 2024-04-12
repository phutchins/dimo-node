package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func InstallIdentityApi(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, SecretsProvider *helm.Chart) (err error) {
	conf := config.New(ctx, "")
	environmentName := conf.Require("environment")

	_, err = apiextensions.NewCustomResource(ctx, "external-secret-identity-api", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("external-secrets.io/v1beta1"),
		Kind:       pulumi.String("ExternalSecret"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("identity-api-secret"),
			Namespace: pulumi.String("identity"),
		},
		OtherFields: map[string]any{
			"spec": map[string]any{
				"secretStoreRef": map[string]any{
					"name": pulumi.String("cluster-secret-store"),
				},
				"target": map[string]any{
					"name": pulumi.String("identity-api-secret"),
				},
				"data": pulumi.Array{
					pulumi.Map{
						"secretKey": pulumi.String("secret"),
						"remoteRef": pulumi.Map{
							"key": pulumi.String("identity-api-secret"),
						},
					},
				},
			},
		},
	}, pulumi.Provider(kubeProvider), pulumi.DependsOn([]pulumi.Resource{SecretsProvider}),
		pulumi.Transformations([]pulumi.ResourceTransformation{
			func(args *pulumi.ResourceTransformationArgs) *pulumi.ResourceTransformationResult {
				if args.Type == "kubernetes:admissionregistration.k8s.io/v1:ValidatingWebhookConfiguration" ||
					args.Type == "kubernetes:admissionregistration.k8s.io/v1:MutatingWebhookConfiguration" {
					return &pulumi.ResourceTransformationResult{
						Props: args.Props,
						Opts: append(args.Opts, pulumi.IgnoreChanges([]string{
							"spec.data",
							"spec.secretStoreRef",
						})),
					}
				}
				return nil
			},
		}),
	)

	if err != nil {
		return err
	}

	//Deploy the users-api from helm chart
	usersApi, err := helm.NewChart(ctx, "identity-api", helm.ChartArgs{
		Chart:     pulumi.String("identity-api"),
		Path:      pulumi.String("./applications/identity-api/charts"),
		Namespace: pulumi.String("identity"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"), // We need to push public versions of the images to docker.io
			},
			"image": pulumi.Map{
				"registry":   pulumi.String("docker.io"),
				"tag":        pulumi.String("latest"),
				"pullPolicy": pulumi.String("IfNotPresent"),
				"repository": pulumi.String("dimozone/identity-api"), // build and push from local for now
			},
			"ingress": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"hosts": pulumi.Array{
					pulumi.Map{
						"host": pulumi.String("identity-api.dimo.zone"), // TODO: Get host from cloud provider
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
			// TODO: Would be nice to have the option to fork from some chain for testing etc... (tenderly)
			// Could use mumbai testnet contracts also in the short term
			// Define sets of addresses at the top level for different environments
			"env": pulumi.Map{
				"KAFKA_BROKERS":                       pulumi.String("kafka-prod-dimo-kafka-kafka-brokers:9092"),
				"DIMO_REGISTRY_CHAIN_ID":              pulumi.Int(137),
				"DIMO_REGISTRY_ADDR":                  pulumi.String("0xFA8beC73cebB9D88FF88a2f75E7D7312f2Fd39EC"),
				"DIMO_VEHICLE_NFT_ADDR":               pulumi.String("0xbA5738a18d83D41847dfFbDC6101d37C69c9B0cF"),
				"AFTERMARKET_DEVICE_CONTRACT_ADDRESS": pulumi.String("0x9c94C395cBcBDe662235E0A9d3bB87Ad708561BA"),
				"DCN_REGISTRY_ADDR":                   pulumi.String("0xE9F4dfE02f895DC17E2e146e578873c9095bA293"),
				"DCN_RESOLVER_ADDR":                   pulumi.String("0x60627326F55054Ea448e0a7BC750785bD65EF757"),
				"SYNTHETIC_DEVICE_CONTRACT_ADDRESS":   pulumi.String("0x4804e8D1661cd1a1e5dDdE1ff458A7f878c0aC6D"),
				"REWARDS_CONTRACT_ADDRESS":            pulumi.String("0x375885164266d48C48abbbb439Be98864Ae62bBE"),
				"BASE_IMAGE_URL":                      pulumi.String("https://devices-api.dimo.zone/v1"),
			},
			"kafka": pulumi.Map{
				"clusterName": pulumi.String("kafka-" + environmentName + "-dimo-kafka"), // TODO: Make this a configurable env value
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("identityAPI", usersApi.URN())

	return nil
}
