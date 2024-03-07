package applications

import (
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/secretmanager"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallDexAuthN(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, SecretsProvider *helm.Chart) (err error) {
	//conf := config.New(ctx, "")
	//environmentName := conf.Require("environment")

	// Create a secret for the dex-auth-n called dex-apple-auth-secret

	// Do it with the external secrets provider crd instead of through google?
	_, err = apiextensions.NewCustomResource(ctx, "external-secret-dex-auth-n", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("external-secrets.io/v1beta1"),
		Kind:       pulumi.String("ExternalSecret"),
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("dex-apple-auth-secret"),
			Namespace: pulumi.String("dex"),
		},
		OtherFields: map[string]interface{}{
			"key":  pulumi.String("secret"),
			"name": pulumi.String("daas-secret"),
		},
	}, pulumi.Provider(kubeProvider), pulumi.DependsOn([]pulumi.Resource{SecretsProvider}))
	if err != nil {
		return err
	}

	daas, err := secretmanager.NewSecret(ctx, "dex-apple-auth-secret", &secretmanager.SecretArgs{
		Labels: pulumi.StringMap{
			"label": pulumi.String("dex-auth-n-secret"),
		},
		Replication: &secretmanager.SecretReplicationArgs{
			UserManaged: &secretmanager.SecretReplicationUserManagedArgs{
				Replicas: secretmanager.SecretReplicationUserManagedReplicaArray{
					&secretmanager.SecretReplicationUserManagedReplicaArgs{
						Location: pulumi.String("us-central1"),
					},
					&secretmanager.SecretReplicationUserManagedReplicaArgs{
						Location: pulumi.String("us-east1"),
					},
				},
			},
		},
		SecretId: pulumi.String("secret"),
	}, pulumi.Provider(kubeProvider), pulumi.DependsOn([]pulumi.Resource{}))
	if err != nil {
		return err
	}

	// Create a secret version for the dex-auth-n called daas-secret-version
	_, err = secretmanager.NewSecretVersion(ctx, "daas-secret-version", &secretmanager.SecretVersionArgs{
		Secret:     pulumi.String("dex-apple-auth-secret"),
		SecretData: pulumi.String("secret-data"),
	}, pulumi.Provider(kubeProvider), pulumi.DependsOn([]pulumi.Resource{daas}))
	if err != nil {
		return err
	}

	//return nil

	/*
		secret_basic, err := secretmanager.NewSecret(ctx, "secret-basic", &secretmanager.SecretArgs{
			SecretId: pulumi.String("secret-version"),
			Labels: pulumi.StringMap{
				"label": pulumi.String("my-label"),
			},
			Replication: &secretmanager.SecretReplicationArgs{
				UserManaged: &secretmanager.SecretReplicationUserManagedArgs{
					Replicas: secretmanager.SecretReplicationUserManagedReplicaArray{
						&secretmanager.SecretReplicationUserManagedReplicaArgs{
							Location: pulumi.String("us-central1"),
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		_, err = secretmanager.NewSecretVersion(ctx, "secret-version-basic", &secretmanager.SecretVersionArgs{
			Secret:     secret_basic.ID(),
			SecretData: pulumi.String("secret-data"),
		})
		if err != nil {
			return err
		}
	*/

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
	// 			"pullPolicy": pulumi.String("IfNotPresent"),
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
