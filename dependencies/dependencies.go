package dependencies

import (
	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	networkingv1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/networking/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

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

	return err
}

func InstallDependencies(ctx *pulumi.Context, provider *kubernetes.Provider) (error, *helm.Chart) {
	// Install external-secrets operator and get the ClusterSecretStore
	secretsProvider, err := InstallSecretsDependencies(ctx, provider)
	if err != nil {
		return err, nil
	}

	// Install nginx-ingress
	if err := InstallNginxIngress(ctx, provider); err != nil {
		return err, nil
	}

	// Install cert-manager and configure Let's Encrypt
	if err := InstallLetsEncrypt(ctx, provider); err != nil {
		return err, nil
	}

	// Create Grafana ingress
	if err := createGrafanaIngress(ctx, provider); err != nil {
		return err, nil
	}

	return nil, secretsProvider
}

func InstallNginxIngress(ctx *pulumi.Context, provider *kubernetes.Provider) error {
	// Create namespace for nginx-ingress
	err := utils.CreateNamespaces(ctx, provider, []string{"ingress-nginx"})
	if err != nil {
		return err
	}

	// Install main nginx-ingress controller
	_, err = helm.NewChart(ctx, "ingress-nginx", helm.ChartArgs{
		Chart: pulumi.String("ingress-nginx"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://kubernetes.github.io/ingress-nginx"),
		},
		Version:   pulumi.String("4.11.2"),
		Namespace: pulumi.String("ingress-nginx"),
		Values: pulumi.Map{
			"controller": pulumi.Map{
				"image": pulumi.Map{
					"chroot": pulumi.Bool(true),
				},
				"kind":         pulumi.String("Deployment"),
				"replicaCount": pulumi.Int(2),
				"ingressClassResource": pulumi.Map{
					"name":            pulumi.String("nginx"),
					"enabled":         pulumi.Bool(true),
					"default":         pulumi.Bool(false),
					"controllerValue": pulumi.String("k8s.io/ingress-nginx"),
				},
				"metrics": pulumi.Map{
					"enabled": pulumi.Bool(true),
					"serviceMonitor": pulumi.Map{
						"enabled": pulumi.Bool(true),
					},
				},
				"resources": pulumi.Map{
					"requests": pulumi.Map{
						"cpu":    pulumi.String("100m"),
						"memory": pulumi.String("200Mi"),
					},
					"limits": pulumi.Map{
						"cpu":    pulumi.String("500m"),
						"memory": pulumi.String("500Mi"),
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
