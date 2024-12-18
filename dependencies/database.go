package dependencies

import (
	"github.com/dimo/dimo-node/infrastructure"
	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Define variables needed globally in the dependencies package

func InstallDatabaseDependencies(ctx *pulumi.Context) (err error) {
	_, err = utils.CreateNamespaces(ctx, infrastructure.KubeProvider, []string{"postgres"})
	if err != nil {
		return err
	}

	// Create a secret for the PostgreSQL password
	_, err = corev1.NewSecret(ctx, "dimoapp-password", &corev1.SecretArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("dimo-postgres-cluster-pguser-dimoapp"),
			Namespace: pulumi.String("postgres"),
		},
		StringData: pulumi.StringMap{
			"verifier": pulumi.String("SCRAM-SHA-256$4096:w]5lKc-/F-Cja^ew@01Ror_,%"),
		},
	}, pulumi.Provider(infrastructure.KubeProvider))
	if err != nil {
		return err
	}

	// Deploy the postgres-operator Helm chart.
	postgresOperatorChart, err := helm.NewRelease(ctx, "postgres-operator", &helm.ReleaseArgs{
		Chart: pulumi.String("./dependencies/charts/postgres-operator"),
		ValueYamlFiles: pulumi.AssetOrArchiveArray{
			pulumi.NewFileAsset("./dependencies/charts/postgres-operator/values.yaml"),
		},
		Namespace: pulumi.String("postgres"),
		Version:   pulumi.String("5.5.1"), // replace with the desired chart version
	}, pulumi.Provider(infrastructure.KubeProvider))
	if err != nil {
		return err
	}

	// Define a PostgresCluster resource after the Postgres Operator has been deployed.
	_, err = apiextensions.NewCustomResource(ctx, "postgres-cluster", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("postgres-operator.crunchydata.com/v1beta1"),
		Kind:       pulumi.String("PostgresCluster"),
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("dimo-postgres-cluster"),
		},
		OtherFields: map[string]any{
			"spec": map[string]any{
				"postgresVersion": pulumi.Int(16),
				"instances": pulumi.Array{
					pulumi.Map{
						"name":     pulumi.String("instance1"),
						"replicas": pulumi.Int(2),
						"dataVolumeClaimSpec": pulumi.Map{
							"storageClassName": pulumi.String("standard"),
							"accessModes":      pulumi.StringArray{pulumi.String("ReadWriteOnce")},
							"resources": pulumi.Map{
								"requests": pulumi.Map{
									"storage": pulumi.String("1Gi"),
								},
							},
						},
					},
				},
				"users": pulumi.Array{
					pulumi.Map{
						"name": pulumi.String("dimoapp"),
						"databases": pulumi.Array{
							pulumi.String("dimoapp"),
						},
						"options": pulumi.String("CREATEDB"),
					},
				},
				"port": pulumi.Int(5432),
				"backups": map[string]any{
					"pgbackrest": pulumi.Map{
						"image": pulumi.String("registry.developers.crunchydata.com/crunchydata/crunchy-pgbackrest:ubi8-2.45-2"),
						"repos": pulumi.Array{
							pulumi.Map{
								"name": pulumi.String("repo1"),
								"volume": pulumi.Map{
									"volumeClaimSpec": pulumi.Map{
										"accessModes": pulumi.StringArray{pulumi.String("ReadWriteOnce")},
										"resources": pulumi.Map{
											"requests": pulumi.Map{
												"storage": pulumi.String("1Gi"),
											},
										},
									},
								},
							},
						},
					},
				},
			},
			// Define other properties like storage, backups, and user configuration.
		},
	}, pulumi.DependsOn([]pulumi.Resource{postgresOperatorChart}), pulumi.Provider(infrastructure.KubeProvider))
	if err != nil {
		return err
	}

	// Export the name of the cluster
	ctx.Export("dimoPGCluster", pulumi.String("dimo-postgres-cluster"))
	return nil

	// Maybe use the crunchyroll operator instead of the zalando one
	// https://www.pulumi.com/ai/answers/s7UarM5px7wLQgmwWRmrMB/orchestrating-postgresql-on-kubernetes-with-crunchydatas-operator
	// https://access.crunchydata.com/documentation/postgres-operator/latest
	// Deploy the postgres operator with helm chart

	// postgresOperator, err := helm.NewChart(ctx, "postgres-operator", helm.ChartArgs{
	// 	Chart: pulumi.String("postgres-operator"),
	// 	FetchArgs: helm.FetchArgs{
	// 		Repo: pulumi.String("https://opensource.zalando.com/postgres-operator/charts/postgres-operator"),
	// 	},
	// 	Namespace: pulumi.String("postgres"),
	// 	Values: pulumi.Map{
	// 		"global": pulumi.Map{
	// 			"imageRegistry": pulumi.String("docker.io"),
	// 		},
	// 		"operator": pulumi.Map{
	// 			"image": pulumi.Map{
	// 				"repository": pulumi.String("bitnami/postgres-operator"),
	// 				"tag":        pulumi.String("1.7.0-debian-10-r0"),
	// 			},
	// 		},
	// 	},
	// }, pulumi.Provider(infrastructure.KubeProvider))
	// if err != nil {
	// 	return err
	// }

	// ctx.Export("postgresOperator", postgresOperator.URN())

	// // Deploy the postgres operator ui helm chart
	// postgresOperatorUI, err := helm.NewChart(ctx, "postgres-operator-ui", helm.ChartArgs{
	// 	Chart: pulumi.String("postgres-operator-ui"),
	// 	FetchArgs: helm.FetchArgs{
	// 		Repo: pulumi.String("https://opensource.zalando.com/postgres-operator/charts/postgres-operator-ui"),
	// 	},
	// 	Namespace: pulumi.String("postgres"),
	// 	Values: pulumi.Map{
	// 		/*
	// 			"global": pulumi.Map{
	// 				"imageRegistry": pulumi.String("docker.io"),
	// 			},
	// 			"image": pulumi.Map{
	// 				"registry": pulumi.String("docker.io"),
	// 				"tag":      pulumi.String("1.7.0-debian-10-r0"),
	// 			}, */
	// 		"service": pulumi.Map{
	// 			"type": pulumi.String("ClusterIP"),
	// 		},
	// 	},
	// }, pulumi.Provider(infrastructure.KubeProvider))
	// if err != nil {
	// 	return err
	// }

	// ctx.Export("postgresOperatorUI", postgresOperatorUI.URN())

	// // Deploy a postgres cluster with a custom resource definition
	// postgresClusterCRD, err := yaml.NewConfigFile(ctx, "postgres-cluster-crd", &yaml.ConfigFileArgs{
	// 	File: string("./dependencies/crds/postgres-cluster-crd.yaml"),
	// }, pulumi.Provider(infrastructure.KubeProvider))
	// if err != nil {
	// 	return err
	// }

	// ctx.Export("postgresClusterCRD", postgresClusterCRD.URN())

	/*
		_, err = apiextensions.NewCustomResource(ctx, "acid-minimal-cluster", &apiextensions.CustomResourceArgs{
			ApiVersion: pulumi.String("acid.zalan.do/v1"),
			Kind:       pulumi.String("postgresql"),
			Metadata: metav1.ObjectMetaArgs{
				Name:      pulumi.String("acid-minimal-cluster"),
				Namespace: pulumi.String("postgres"),
			},
			OtherFields: map[string]interface{}{
				"spec": pulumi.Map{
					"teamId":            pulumi.String("acid"),
					"volume":            pulumi.Map{"size": pulumi.String("1Gi"), "storageClass": pulumi.String("standard")},
					"numberOfInstances": pulumi.Int(2),
					"users":             pulumi.Map{"zalando": pulumi.Array{}},
					"databases":         pulumi.Map{"foo": pulumi.String("zalando")},
					"postgresql": pulumi.Map{
						"version": pulumi.String("13"),
					},
				},
			},
			// TODO: Add NS dependency back to this: pulumi.DependsOn([]pulumi.Resource{ns})
		}, pulumi.Provider(infrastructure.KubeProvider))
		if err != nil {
			return err
		}
	*/
}
