package applications

import (
	"slices"

	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallApplications(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, SecretsProvider *helm.Chart) (err error) {
	// Use this later to configure sets of applications to install
	applications := []string{
		//"users-api",
		//"prometheus",
		"kube-prometheus-stack",
		//"identity-api",
		//"device-data-api",
		//"contract-event-processor",
		//"mqtt-broker",
		//"dex-auth-n", // Authentication
		//"dex-auth-z", // Authorization
		//"webhook-validator",
		//"certificate-authority",
		//"dex",
	}

	// Create namespaces for applications
	namespaceMap, err := utils.CreateNamespaces(ctx, kubeProvider, []string{"device-data", "users", "identity-api", "monitoring"})
	if err != nil {
		return err
	}

	if slices.Contains(applications, "kube-prometheus-stack") {
		err = InstallKubePrometheus(ctx, kubeProvider, namespaceMap["monitoring"])
		if err != nil {
			return err
		}
	}

	// // Install Prometheus
	// if slices.Contains(applications, "prometheus") {
	// 	err = InstallPrometheus(ctx, kubeProvider)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// Users API - https://github.com/DIMO-Network/users-api/tree/main/charts/users-api
	// Chart Link [ ]
	//
	if slices.Contains(applications, "users-api") {
		err = InstallUsersApi(ctx, kubeProvider)
		if err != nil {
			return err
		}
	}

	// Identity API
	if slices.Contains(applications, "identity-api") {
		err = InstallIdentityApi(ctx, kubeProvider, SecretsProvider)
		if err != nil {
			return err
		}
	}

	// Device Data API
	if slices.Contains(applications, "device-data-api") {
		err = InstallDeviceDataApi(ctx, kubeProvider, SecretsProvider)
		if err != nil {
			return err
		}
	}

	// Contract Event Processor
	if slices.Contains(applications, "contract-event-processor") {
		err = InstallContractEventProcessor(ctx, kubeProvider)
		if err != nil {
			return err
		}
	}

	// MQTT Broker - Cluster Helm Charts (single instance, two services inside)
	// May need to create a config map with bogus pub/priv keypair for now
	// Pull most everything in
	// Will need to set up ingress
	if slices.Contains(applications, "mqtt-broker") {
		err = InstallMQTTBroker(ctx, kubeProvider)
		if err != nil {
			return err
		}
	}

	// Dex - Cluster Helm Charts (two sets of values files, values/values-prod (dex1 - Auth N) )
	// roles-rights/roles-rights-prod (dex2 - Auth Z) - turns auth token into vehicle auth token

	// Dex1 - Auth N
	// Static clients block will go away and move to on chain (try to skip it for now)
	// issuer is just URL config (issued by)
	// Create the dex-X-secret (dont include environment from and don't create)
	if slices.Contains(applications, "dex-auth-n") {
		err = InstallDexAuthN(ctx, kubeProvider, SecretsProvider)
		if err != nil {
			return err
		}
	}

	// Dex2 - Auth Z
	// Token expires quicker than the other (10 min)
	// Not exposed publicly
	// Connector may not be necessary (even though it says it is lol)
	if slices.Contains(applications, "dex-auth-z") {
		err = InstallDexAuthZ(ctx, kubeProvider, SecretsProvider)
		if err != nil {
			return err
		}
	}

	// token-exchange-api
	//   ENVIRONMENT: prod
	//   JWT_KEY_SET_URL: https://auth.dimo.zone/keys
	//   DEX_GRPC_ADDRESS: dex-roles-rights-prod:5557
	//   USERS_API_GRPC_ADDRESS: users-api-prod:8086
	//   CONTRACT_ADDRESS_WHITELIST: '0xba5738a18d83d41847dffbdc6101d37c69c9b0cf'
	//     - address is vehicle contract address

	// Webhook Validator
	// Token Base Uri is used in the certificateResponseData
	// Already configured with chain_id 137 (polygon)
	if slices.Contains(applications, "webhook-validator") {
		err = InstallWebhookValidator(ctx, kubeProvider)
		if err != nil {
			return err
		}
	}

	// Certificate Authority
	// Before needed to manually create KMS keys (generate) - There is a GIST for this but now is a CLI
	//   ^ for aws, need to figure out GCP
	if slices.Contains(applications, "certificate-authority") {
		err = InstallCertificateAuthority(ctx, kubeProvider)
		if err != nil {
			return err
		}
	}

	return nil
}
