package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	v1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallPrometheus(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	_, err = helm.NewChart(ctx, "prometheus", helm.ChartArgs{
		Chart: pulumi.String("prometheus"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://prometheus-community.github.io/helm-charts"),
		},
		Namespace: pulumi.String("monitoring"),
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	// Expose Prometheus server using a LoadBalancer Service
	_, err = v1.NewService(ctx, "prometheus-service", &v1.ServiceArgs{
		Metadata: &v1.ObjectMetaArgs{
			Name: pulumi.String("prometheus-service"),
		},
		Spec: &v1.ServiceSpecArgs{
			Type: pulumi.String("LoadBalancer"),
			Selector: pulumi.StringMap{
				"app": pulumi.String("prometheus"),
			},
			Ports: v1.ServicePortArray{
				&v1.ServicePortArgs{
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(9090), // Prometheus server default port
				},
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
