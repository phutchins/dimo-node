package dependencies

import (
	"github.com/dimo/dimo-node/infrastructure"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Define variables needed globally in the dependencies package

func InstallLinkerD(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	// Install linkerd helm chart
	linkerd, err := helm.NewChart(ctx, "linkerd", helm.ChartArgs{
		Chart: pulumi.String("linkerd2"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://helm.linkerd.io/stable"),
		},
		Namespace: pulumi.String("dimo"),
		Values: pulumi.Map{
			"global": pulumi.Map{
				"imageRegistry": pulumi.String("docker.io"),
			},
			"controllerImage": pulumi.Map{
				"imagePullPolicy": pulumi.String("IfNotPresent"),
			},
			"proxy": pulumi.Map{
				"image": pulumi.Map{
					"imagePullPolicy": pulumi.String("IfNotPresent"),
				},
			},
			"identity": pulumi.Map{
				"trustDomain": pulumi.String("cluster.local"),
			},
			"installNamespace": pulumi.Bool(true),
			"jaeger": pulumi.Map{
				"image": pulumi.Map{
					"imagePullPolicy": pulumi.String("IfNotPresent"),
				},
			},
			"grafana": pulumi.Map{
				"image": pulumi.Map{
					"imagePullPolicy": pulumi.String("IfNotPresent"),
				},
			},
			"prometheus": pulumi.Map{
				"image": pulumi.Map{
					"imagePullPolicy": pulumi.String("IfNotPresent"),
				},
			},
			"web": pulumi.Map{
				"image": pulumi.Map{
					"imagePullPolicy": pulumi.String("IfNotPresent"),
				},
			},
		},
	}, pulumi.Provider(infrastructure.KubeProvider))
	if err != nil {
		return err
	}

	ctx.Export("linkerd", linkerd.URN())

	return nil
}
