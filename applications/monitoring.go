package applications

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallKubePrometheus(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, namespace *corev1.Namespace) (err error) {
	_, err = helm.NewRelease(ctx, "kube-prometheus-stack", &helm.ReleaseArgs{
		Name:    pulumi.String("kube-prometheus-stack"),
		Chart:   pulumi.String("kube-prometheus-stack"),
		Version: pulumi.String("66.3.0"),
		RepositoryOpts: &helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://prometheus-community.github.io/helm-charts"),
		},
		Namespace:       pulumi.String("monitoring"),
		CreateNamespace: pulumi.Bool(true),
		SkipAwait:       pulumi.Bool(true),
		WaitForJobs:     pulumi.Bool(false),
		Replace:         pulumi.Bool(true),
		Values: pulumi.Map{
			"grafana": pulumi.Map{
				"admin": pulumi.Map{
					"existingSecret": pulumi.String("grafana-password-secret"),
					"passwordKey":    pulumi.String("password"),
				},
			},
		},
	}, pulumi.Provider(kubeProvider), pulumi.DependsOn([]pulumi.Resource{namespace}))
	if err != nil {
		return err
	}

	// Create Grafana ingress
	if err := createGrafanaIngress(ctx, kubeProvider); err != nil {
		return err
	}

	return nil
}

func createGrafanaIngress(ctx *pulumi.Context, provider *kubernetes.Provider) error {
	_, err := networkingv1.NewIngress(ctx, "grafana-ingress", &networkingv1.IngressArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name:      pulumi.String("grafana"),
			Namespace: pulumi.String("monitoring"),
			Annotations: pulumi.StringMap{
				"nginx.ingress.kubernetes.io/ssl-redirect": pulumi.String("true"),
				"cert-manager.io/cluster-issuer":           pulumi.String("letsencrypt-prod"),
			},
		},
		Spec: &networkingv1.IngressSpecArgs{
			IngressClassName: pulumi.String("nginx"),
			Tls: networkingv1.IngressTLSArray{
				&networkingv1.IngressTLSArgs{
					Hosts:      pulumi.StringArray{pulumi.String("monitoring.driveomid.xyz")},
					SecretName: pulumi.String("grafana-tls"),
				},
			},
			Rules: networkingv1.IngressRuleArray{
				&networkingv1.IngressRuleArgs{
					Host: pulumi.String("monitoring.driveomid.xyz"),
					Http: &networkingv1.HTTPIngressRuleValueArgs{
						Paths: networkingv1.HTTPIngressPathArray{
							&networkingv1.HTTPIngressPathArgs{
								Path:     pulumi.String("/"),
								PathType: pulumi.String("Prefix"),
								Backend: &networkingv1.IngressBackendArgs{
									Service: &networkingv1.IngressServiceBackendArgs{
										Name: pulumi.String("kube-prometheus-stack-grafana"),
										Port: &networkingv1.ServiceBackendPortArgs{
											Number: pulumi.Int(80),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	return nil
}

// func InstallPrometheus(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
// 	ctx.Log.Info("Installing Prometheus chart...", nil)
// 	_, err = helm.NewChart(ctx, "prometheus", helm.ChartArgs{
// 		Chart: pulumi.String("prometheus"),
// 		FetchArgs: helm.FetchArgs{
// 			Repo: pulumi.String("https://prometheus-community.github.io/helm-charts"),
// 		},
// 		Namespace: pulumi.String("monitoring"),
// 		Values: pulumi.Map{
// 			"rbac": pulumi.Map{
// 				"create": pulumi.Bool(true),
// 			},
// 			"serviceAccounts": pulumi.Map{
// 				"alertmanager": pulumi.Map{
// 					"create": pulumi.Bool(true),
// 				},
// 				"nodeExporter": pulumi.Map{
// 					"create": pulumi.Bool(true),
// 				},
// 				"pushgateway": pulumi.Map{
// 					"create": pulumi.Bool(true),
// 				},
// 				"server": pulumi.Map{
// 					"create": pulumi.Bool(true),
// 				},
// 			},
// 		},
// 	}, pulumi.Provider(kubeProvider))
// 	if err != nil {
// 		return err
// 	}

// 	// Expose Prometheus server using a LoadBalancer Service
// 	_, err = corev1.NewService(ctx, "prometheus-service", &corev1.ServiceArgs{
// 		Metadata: &metav1.ObjectMetaArgs{
// 			Name:      pulumi.String("prometheus-service"),
// 			Namespace: pulumi.String("monitoring"),
// 		},
// 		Spec: &corev1.ServiceSpecArgs{
// 			Type: pulumi.String("LoadBalancer"),
// 			Selector: pulumi.StringMap{
// 				"app": pulumi.String("prometheus"),
// 			},
// 			Ports: corev1.ServicePortArray{
// 				&corev1.ServicePortArgs{
// 					Port:       pulumi.Int(80),
// 					TargetPort: pulumi.Int(9090), // Prometheus server default port
// 				},
// 			},
// 		},
// 	}, pulumi.Provider(kubeProvider))
// 	if err != nil {
// 		return err
// 	}

// 	// ctx.Export("prometheus-service-url", service.Status.ApplyT(func(status *corev1.ServiceStatus) (string, error) {
// 	// 	return status.LoadBalancer.Ingress[0].Hostname, nil
// 	// }))

// 	//ctx.Export("prometheus-service", service.Metadata.Elem().Name())

// 	return nil
// }

// func InstallGrafana(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
// 	ctx.Log.Info("Installing Grafana chart...", nil)
// 	_, err = helm.NewChart(ctx, "grafana", helm.ChartArgs{
// 		Chart: pulumi.String("grafana"),
// 		FetchArgs: helm.FetchArgs{
// 			Repo: pulumi.String("https://grafana.github.io/helm-charts"),
// 		},
// 		Namespace: pulumi.String("monitoring"),
// 		Values: pulumi.Map{
// 			"adminPassword": pulumi.String("admin"), // You should use a secret in production
// 			"datasources": pulumi.Map{
// 				"datasources.yaml": pulumi.Map{
// 					"apiVersion": pulumi.Int(1),
// 					"datasources": pulumi.Array{
// 						pulumi.Map{
// 							"name":      pulumi.String("Prometheus"),
// 							"type":      pulumi.String("prometheus"),
// 							"url":       pulumi.String("http://prometheus-server.monitoring.svc.cluster.local"),
// 							"access":    pulumi.String("proxy"),
// 							"isDefault": pulumi.Bool(true),
// 						},
// 					},
// 				},
// 			},
// 			"service": pulumi.Map{
// 				"type": pulumi.String("LoadBalancer"),
// 			},
// 		},
// 	}, pulumi.Provider(kubeProvider))
// 	return err
// }
