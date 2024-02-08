package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func InstallDeviceDataApi(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	conf := config.New(ctx, "")
	environmentName := conf.Require("environment")
	//Deploy the users-api from helm chart
	usersApi, err := helm.NewChart(ctx, "device-data-api", helm.ChartArgs{
		Chart: pulumi.String("device-data-api"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://dimo-network.github.io/device-data-api"),
		},
		Namespace: pulumi.String("device-data"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"), // We need to push public versions of the images to docker.io
			},
			"image": pulumi.Map{
				"registry":   pulumi.String("docker.io"),
				"tag":        pulumi.String("latest"),
				"pullPolicy": pulumi.String("IfNotPResent"),
				"repository": pulumi.String("dimo-network/device-data-api"), // build and push from local for now
			},
			"ingress": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"annotations": pulumi.Map{
					"nginx.ingress.kubernetes.io/auth-tls-secret":        pulumi.String("ingress/cf-origin-ca"),
					"nginx.ingress.kubernetes.io/auth-tls-verify-client": pulumi.String("on"),
					"nginx.ingress.kubernetes.io/enable-cors":            pulumi.String("true"),
					"nginx.ingress.kubernetes.io/cors-allow-origin":      pulumi.String("https://app.dimo.zone"),
					"nginx.ingress.kubernetes.io/limit-rps":              pulumi.String("9"),
					"external-dns.alpha.kubernetes.io/hostname":          pulumi.String("device-data-api.dimo.zone"),
				},
				"hosts": pulumi.Array{
					pulumi.Map{
						"host": pulumi.String("device-data-api.dimo.zone"), // TODO: Get host from cloud provider
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
				"ENVIRONMENT":                       pulumi.String(environmentName),
				"PORT":                              pulumi.String("8080"),
				"LOG_LEVEL":                         pulumi.String("info"),
				"SERVICE_NAME":                      pulumi.String("device-data-api"), // ?
				"JWT_KEY_SET_URL":                   pulumi.String("https://auth.dimo.zone/keys"), // Comes from DEX
				"DEPLOYMENT_BASE_URL":               pulumi.String("https://device-data-api.dimo.zone"),
				"DEVICE_DATA_INDEX_NAME":            pulumi.String("device-status-prod*"),
				"DEVICE_DATA_INDEX_NAME_V2":         pulumi.String("vss-status-prod*"),
				"DEVICES_APIGRPC_ADDR":              pulumi.String("devices-api-prod:8086"),
				"ENABLE_PRIVILEGES":                 pulumi.Bool(true), // Should prob always be true
				"TOKEN_EXCHANGE_JWK_KEY_SET_URL":    pulumi.String("http://dex-roles-rights-" + environmentName + ".prod.svc.cluster.local:5556/keys"), // TODO: Replace this other prod with cluster url stuff
				"VEHICLE_NFT_ADDRESS":               pulumi.String("0xba5738a18d83d41847dffbdc6101d37c69c9b0cf"), // Might not need to be here later?
				//"DEVICE_DEFINITIONS_GRPC_ADDR":      pulumi.String("device-definitions-api-prod:8086"), // See if service works without this
				"USERS_API_GRPC_ADDR":               pulumi.String("users-api-prod:8086"),
				//"AWS_BUCKET_NAME":                   pulumi.String("dimo-network-device-data-export-prod"), // Needs to go away
				//"NATS_URL":                          pulumi.String("nats-prod:4222"), // Why?
				"KAFKA_BROKERS":                     pulumi.String("kafka-" + environmentName + "-dimo-kafka-kafka-brokers:9092"),
				"DEVICE_FINGERPRINT_TOPIC":          pulumi.String("topic.device.fingerprint"),
				"DEVICE_FINGERPRINT_CONSUMER_GROUP": pulumi.String("device-fingerprint-vin-data"),
			},
			"job": pulumi.Map{ // Disable this for now!
				"name":     pulumi.String("generate-report-vehicle-signals-event"),
				"schedule": pulumi.String("0 0 * * 0"),
				"args": pulumi.Array{
					pulumi.String("-c"),
					pulumi.String("/device-data-api generate-report-vehicle-signals; CODE=$?; echo \"weekly vehicle data dashboard report\"; wget -q --post-data \"hello=shutdown\" http://localhost:4191/shutdown &> /dev/null; exit $CODE;"),
				},
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("deviceDataAPI", usersApi.URN())

	return nil
}
