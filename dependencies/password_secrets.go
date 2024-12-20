package dependencies

import (
	"encoding/json"
	"fmt"

	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

// ManagePasswordSecrets creates or updates External Secrets for passwords
func ManagePasswordSecrets(ctx *pulumi.Context, provider *kubernetes.Provider, clusterSecretStore *apiextensions.CustomResource) error {
	// Get password configurations from stack config
	conf := config.New(ctx, "")
	configs := conf.Get("password-configs")
	if configs == "" {
		return nil // No configurations found, nothing to do
	}

	var passwordConfigs map[string]utils.PasswordConfig
	if err := json.Unmarshal([]byte(configs), &passwordConfigs); err != nil {
		return fmt.Errorf("failed to parse password configs: %v", err)
	}

	// Create External Secret for each password configuration
	for _, config := range passwordConfigs {
		_, err := apiextensions.NewCustomResource(ctx, fmt.Sprintf("%s-external-secret", config.ServiceName),
			&apiextensions.CustomResourceArgs{
				ApiVersion: pulumi.String("external-secrets.io/v1beta1"),
				Kind:       pulumi.String("ExternalSecret"),
				Metadata: &metav1.ObjectMetaArgs{
					Name:      pulumi.String(config.K8sSecretName),
					Namespace: pulumi.String(config.K8sNamespace),
				},
				OtherFields: map[string]interface{}{
					"spec": map[string]interface{}{
						"refreshInterval": "1h",
						"secretStoreRef": map[string]interface{}{
							"name": "cluster-secret-store",
							"kind": "ClusterSecretStore",
						},
						"target": map[string]interface{}{
							"name":           config.K8sSecretName,
							"creationPolicy": "Owner",
						},
						"data": []map[string]interface{}{
							{
								"secretKey": "password",
								"remoteRef": map[string]interface{}{
									"key": config.GCPSecretID,
								},
							},
						},
					},
				},
			},
			pulumi.Provider(provider),
			pulumi.DependsOn([]pulumi.Resource{clusterSecretStore}),
		)
		if err != nil {
			return fmt.Errorf("failed to create external secret for %s: %v", config.ServiceName, err)
		}
	}

	return nil
}
