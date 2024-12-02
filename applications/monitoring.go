package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	//"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallPrometheus(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	_, err = helm.NewChart(ctx, "prometheus", helm.ChartArgs{
		Chart: pulumi.String("prometheus"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://prometheus-community.github.io/helm-charts"),
		},
		Namespace: pulumi.String("monitoring"),
		Values: pulumi.Map{
			"rbac": pulumi.Map{
				"create": pulumi.Bool(true),
			},
			"serviceAccounts": pulumi.Map{
				"alertmanager": pulumi.Map{
					"create": pulumi.Bool(true),
				},
				"nodeExporter": pulumi.Map{
					"create": pulumi.Bool(true),
				},
				"pushgateway": pulumi.Map{
					"create": pulumi.Bool(true),
				},
				"server": pulumi.Map{
					"create": pulumi.Bool(true),
				},
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	// Expose Prometheus server using a LoadBalancer Service
	_, err = corev1.NewService(ctx, "prometheus-service", &corev1.ServiceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("prometheus-service"),
			Namespace: pulumi.String("monitoring"),
		},
		Spec: &corev1.ServiceSpecArgs{
			Type: pulumi.String("LoadBalancer"),
			Selector: pulumi.StringMap{
				"app": pulumi.String("prometheus"),
			},
			Ports: corev1.ServicePortArray{
				&corev1.ServicePortArgs{
					Port:       pulumi.Int(80),
					TargetPort: pulumi.Int(9090), // Prometheus server default port
				},
			},
		},
	}, pulumi.Provider(kubeProvider))
	if err != nil {
		return err
	}

	// ctx.Export("prometheus-service-url", service.Status.ApplyT(func(status *corev1.ServiceStatus) (string, error) {
	// 	return status.LoadBalancer.Ingress[0].Hostname, nil
	// }))

	//ctx.Export("prometheus-service", service.Metadata.Elem().Name())

	return nil
}

func InstallKubePrometheus(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	_, err = helm.NewChart(ctx, "kube-prometheus-stack", helm.ChartArgs{
		Chart:   pulumi.String("kube-prometheus-stack"),
		Version: pulumi.String("66.3.0"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://prometheus-community.github.io/helm-charts"),
		},
		Namespace: pulumi.String("monitoring"),
		Values: pulumi.Map{
			"crds": pulumi.Map{
				"create": pulumi.Bool(true),
			},
			"grafana": pulumi.Map{
				"adminPassword": pulumi.String("admin"),
				"ingress": pulumi.Map{
					"enabled":          pulumi.Bool(true),
					"ingressClassName": pulumi.String("nginx"),
					"hosts": pulumi.Array{
						pulumi.String("grafana.driveomid.xyz"),
					},
					"annotations": pulumi.StringMap{
						"kubernetes.io/ingress.class":    pulumi.String("nginx"),
						"cert-manager.io/cluster-issuer": pulumi.String("letsencrypt-prod"),
					},
					"tls": pulumi.Array{
						pulumi.Map{
							"secretName": pulumi.String("grafana-tls"),
							"hosts": pulumi.Array{
								pulumi.String("grafana.driveomid.xyz"),
							},
						},
					},
				},
			},
			"prometheus": pulumi.Map{
				"serviceAccount": pulumi.Map{
					"create": pulumi.Bool(true),
					"name":   pulumi.String("prometheus"),
				},
				"ingress": pulumi.Map{
					"enabled":          pulumi.Bool(true),
					"ingressClassName": pulumi.String("nginx"),
					"hosts": pulumi.Array{
						pulumi.String("prometheus.driveomid.xyz"),
					},
					"annotations": pulumi.StringMap{
						"kubernetes.io/ingress.class":    pulumi.String("nginx"),
						"cert-manager.io/cluster-issuer": pulumi.String("letsencrypt-prod"),
					},
					"tls": pulumi.Array{
						pulumi.Map{
							"secretName": pulumi.String("prometheus-tls"),
							"hosts": pulumi.Array{
								pulumi.String("prometheus.driveomid.xyz"),
							},
						},
					},
				},
			},
			"rbac": pulumi.Map{
				"create": pulumi.Bool(true),
			},
		},
	}, pulumi.Provider(kubeProvider))
	return err
}

func InstallGrafana(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	_, err = helm.NewChart(ctx, "grafana", helm.ChartArgs{
		Chart: pulumi.String("grafana"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://grafana.github.io/helm-charts"),
		},
		Namespace: pulumi.String("monitoring"),
		Values: pulumi.Map{
			"adminPassword": pulumi.String("admin"), // You should use a secret in production
			"datasources": pulumi.Map{
				"datasources.yaml": pulumi.Map{
					"apiVersion": pulumi.Int(1),
					"datasources": pulumi.Array{
						pulumi.Map{
							"name":      pulumi.String("Prometheus"),
							"type":      pulumi.String("prometheus"),
							"url":       pulumi.String("http://prometheus-server.monitoring.svc.cluster.local"),
							"access":    pulumi.String("proxy"),
							"isDefault": pulumi.Bool(true),
						},
					},
				},
			},
			"service": pulumi.Map{
				"type": pulumi.String("LoadBalancer"),
			},
		},
	}, pulumi.Provider(kubeProvider))
	return err
}
