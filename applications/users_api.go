package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallUsersApi(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	//Deploy the users-api from helm chart
	usersApi, err := helm.NewChart(ctx, "users-api", helm.ChartArgs{
		Chart: pulumi.String("users-api"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://dimo-network.github.io/users-api"),
		},
		Namespace: pulumi.String("users"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"),
			},
			"image": pulumi.Map{
				"registry": pulumi.String("docker.io"),
				"tag":      pulumi.String("latest"),
			},
			"service": pulumi.Map{
				"type": pulumi.String("ClusterIP"),
			},
			"ingress": pulumi.Map{
				"enabled": pulumi.Bool(true),
				"hosts": pulumi.Array{
					pulumi.Map{
						"host": pulumi.String("users-api.dimo.zone"), // TODO: Get host from cloud provider
						"paths": pulumi.Array{
							pulumi.Map{
								"path": pulumi.String("/"),
								"backend": pulumi.Map{
									"serviceName": pulumi.String("users-api"),
									"servicePort": pulumi.String("http"),
								},
							},
						},
					},
				},
			},
			"postgresql": pulumi.Map{
				"enabled":  pulumi.Bool(true),
				"host":     pulumi.String("postgres.postgres.svc.cluster.local"),
				"port":     pulumi.Int(5432),
				"user":     pulumi.String("postgres"),
				"password": pulumi.String("postgres"),
				"database": pulumi.String("postgres"),
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("usersAPI", usersApi.URN())

	return nil
}
