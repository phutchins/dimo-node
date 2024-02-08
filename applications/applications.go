package applications

import (
	"slices"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallApplications(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	// Use this later to configure sets of applications to install
	applications := []string{
		//"users-api",
		"identity-api",
		"device-data-api",
		"contract-event-processor",
		"mqtt-broker",
		"dex",
		"webhook-validator",
		"certificate-authority",
	}

	// Identity API

	// Certificate API

	// MQDT Broker

	// Users API - https://github.com/DIMO-Network/users-api/tree/main/charts/users-api
	if slices.Contains(applications, "users-api") {
		err = InstallUsersApi(ctx, kubeProvider)
		if err != nil {
			return err
		}
	}

	// Identity API
	if slices.Contains(applications, "identity-api") {
		err = InstallIdentityApi(ctx, kubeProvider)
		if err != nil {
			return err
		}
	}

	// Device Data API
	if slices.Contains(applications, "device-data-api") {
		err = InstallDeviceDataApi(ctx, kubeProvider)
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

	/*
		// MQTT Broker - Cluster Helm Charts (single instance, two services inside)
		// May need to create a config map with bogus pub/priv keypair for now
		// Pull most everything in
		// Will need to set up ingress
		if slices.Contains(applications, "mqtt-broker") {
			err = InstallMqttBroker(ctx, kubeProvider)
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
		if slices.Contains(applications, "dex1") {
			err = InstallDex1(ctx, kubeProvider)
			if err != nil {
				return err
			}
		}

		// Dex2 - Auth Z
		if slices.Contains(applications, "dex2") {
			err = InstallDex1(ctx, kubeProvider)
			if err != nil {
				return err
			}
		}

		// Webhook Validator
		if slices.Contains(applications, "webhook-validator") {
			err = InstallWebhookValidator(ctx, kubeProvider)
			if err != nil {
				return err
			}
		}

		// Certificate Authority
		if slices.Contains(applications, "certificate-authority") {
			err = InstallCertificateAuthority(ctx, kubeProvider)
			if err != nil {
				return err
			}
		}

	*/
	return nil
}
