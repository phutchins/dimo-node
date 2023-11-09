package dependencies

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Define variables needed outside the installDependencies() function

func InstallDependencies(ctx *pulumi.Context, KubeConfig pulumi.StringOutput) (err error) {
	// Get kubeconfig from exported to context
	//kubeConfig := stackRef.GetOutput(pulumi.String("kubeConfig"))
	//var getKubeConfig *remote.Command = stackRef.GetOutput(pulumi.String("getKubeConfig"))

	KubeConfig.ApplyT(func(kc string) error {
		fmt.Printf("KubeConfig: %v", kc)
		return nil
	})

	ctx.Export("kubeConfig", KubeConfig)

	kubeProvider, err := kubernetes.NewProvider(ctx, "k3sDeps", &kubernetes.ProviderArgs{
		Kubeconfig: KubeConfig,
	}) // May want to make this do better checking to ensure that the node is all the way up
	if err != nil {
		return err
	}

	// Deploy the postgres operator with helm chart
	postgresOperator, err := helm.NewChart(ctx, "postgres-operator", helm.ChartArgs{
		Chart: pulumi.String("postgres-operator"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://opensource.zalando.com/postgres-operator/charts/postgres-operator"),
		},
		Namespace: pulumi.String("dimo"),
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
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("postgresOperator", postgresOperator.URN())

	// Deploy the postgres cluster with helm chart
	postgresCluster, err := helm.NewChart(ctx, "zalando-postgres-cluster", helm.ChartArgs{
		Chart: pulumi.String("zalando-postgres-cluster"),
		Path:  pulumi.String("./dependencies/charts/"),
		//FetchArgs: helm.FetchArgs{
		//	Repo: pulumi.String("https://charts.bitnami.com/bitnami"),
		//},
		Namespace: pulumi.String("dimo"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"),
			},
			"image": pulumi.Map{
				"registry": pulumi.String("docker.io"),
				"tag":      pulumi.String("11.12.0-debian-10-r0"),
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
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	/*
		// Deploy the postgres operator ui helm chart
		postgresOperatorUI, err := helm.NewChart(ctx, "postgres-operator-ui", helm.ChartArgs{
			Chart: pulumi.String("postgres-operator-ui"),
			FetchArgs: helm.FetchArgs{
				Repo: pulumi.String("https://opensource.zalando.com/postgres-operator/charts/postgres-operator-ui"),
			},
			Namespace: pulumi.String("dimo"),
			Values: pulumi.Map{
				"global": pulumi.Map{
					"imageRegistry": pulumi.String("docker.io"),
				},
				"image": pulumi.Map{
					"registry": pulumi.String("docker.io"),
					"tag":      pulumi.String("1.7.0-debian-10-r0"),
				},
				"service": pulumi.Map{
					"type": pulumi.String("LoadBalancer"),
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("postgresOperatorUI", postgresOperatorUI.URN())
	*/

	ctx.Export("postgresCluster", postgresCluster.URN())

	//func Run(infraCtx *pulumi.Context) *pulumi.Context {
	//var depsCtx *pulumi.Context

	//depsCtx = ctx
	//cts.Ref
	//ctx.StackReference.GetOutput("publicIp")
	//publicIp := accessConfigs.Index(pulumi.Int(0)).NatIp().Elem()
	//fmt.Printf("Postgres Operator URL: ", postgresOperator.URN())

	return nil
}
