package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	//"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func InstallCertificateAuthority(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	//conf := config.New(ctx, "")
	//environmentName := conf.Require("environment")
	//Deploy the users-api from helm chart
	usersApi, err := helm.NewChart(ctx, "certificate-authority", helm.ChartArgs{
		Chart:     pulumi.String("dimo-ca"),
		Path:      pulumi.String("./applications/cluster-helm-charts/charts"),
		Namespace: pulumi.String("certificate-authority"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"), // We need to push public versions of the images to docker.io
			},
			"image": pulumi.Map{
				"registry":   pulumi.String("docker.io"),
				"tag":        pulumi.String("latest"),
				"pullPolicy": pulumi.String("IfNotPresent"),
				"repository": pulumi.String("dimozone/certificate-authority"), // build and push from local for now
			},
			"ingress": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"hosts": pulumi.Array{
					pulumi.Map{
						"host": pulumi.String("certificate-authority.dimo.zone"), // TODO: Get host from cloud provider
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
				"BASE_IMAGE_URL": pulumi.String("https://certificate-authority.dimo.zone/v1"),
			},
		},
	}, pulumi.Provider(kubeProvider),
		pulumi.Transformations([]pulumi.ResourceTransformation{
			func(args *pulumi.ResourceTransformationArgs) *pulumi.ResourceTransformationResult {
				if args.Type == "kubernetes:admissionregistration.k8s.io/v1:ValidatingWebhookConfiguration" ||
					args.Type == "kubernetes:admissionregistration.k8s.io/v1:MutatingWebhookConfiguration" {
					return &pulumi.ResourceTransformationResult{
						Props: args.Props,
						Opts: append(args.Opts, pulumi.IgnoreChanges([]string{
							"spec.data",
						})),
					}
				}
				return nil
			},
		}), pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("CertificateAuthority", usersApi.URN())

	return nil
}
