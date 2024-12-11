package dependencies

import (
	"os/exec"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallLetsEncrypt(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) error {
	// Add cleanup step
	cleanup := exec.Command("kubectl", "delete", "crds",
		"certificaterequests.cert-manager.io",
		"certificates.cert-manager.io",
		"challenges.acme.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"issuers.cert-manager.io",
		"orders.acme.cert-manager.io",
		"--ignore-not-found=true")
	cleanup.Run()

	// Install cert-manager helm chart
	_, err := helm.NewChart(ctx, "cert-manager", helm.ChartArgs{
		Chart:   pulumi.String("cert-manager"),
		Version: pulumi.String("v1.13.2"),
		FetchArgs: helm.FetchArgs{
			Repo: pulumi.String("https://charts.jetstack.io"),
		},
		Namespace: pulumi.String("cert-manager"),
		Values: pulumi.Map{
			"installCRDs": pulumi.Bool(false),
			"global": pulumi.Map{
				"leaderElection": pulumi.Map{
					"namespace": pulumi.String("cert-manager"),
				},
			},
		},
	}, pulumi.Provider(kubeProvider))

	if err != nil {
		return err
	}

	// Create ClusterIssuer for Let's Encrypt
	_, err = apiextensions.NewCustomResource(ctx, "letsencrypt-prod", &apiextensions.CustomResourceArgs{
		ApiVersion: pulumi.String("cert-manager.io/v1"),
		Kind:       pulumi.String("ClusterIssuer"),
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("letsencrypt-prod"),
		},
		OtherFields: map[string]interface{}{
			"spec": map[string]interface{}{
				"acme": map[string]interface{}{
					"server": "https://acme-v02.api.letsencrypt.org/directory",
					"email":  "admin@dimo.zone",
					"privateKeySecretRef": map[string]interface{}{
						"name": "letsencrypt-prod",
					},
					"solvers": []map[string]interface{}{
						{
							"http01": map[string]interface{}{
								"ingress": map[string]interface{}{
									"class": "nginx",
								},
							},
						},
					},
				},
			},
		},
	}, pulumi.Provider(kubeProvider))

	return err
}
