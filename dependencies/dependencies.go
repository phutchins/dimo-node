package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallDependencies(ctx *pulumi.Context, provider *kubernetes.Provider) (error, *kubernetes.Provider) {
	// Create database secrets
	if err := createDatabaseSecrets(ctx, provider); err != nil {
		return err, nil
	}

	// Install external-secrets operator
	SecretsProvider, err := InstallExternalSecrets(ctx, provider)
	if err != nil {
		return err, nil
	}

	// Using this one
	// err = InstallLetsEncrypt(ctx, kubeProvider)
	// if err != nil {
	// 	return err, nil
	// }

	// Not using this one
	// err = InstallCertificateDependencies(ctx, kubeProvider)
	// if err != nil {
	// 	return err, nil
	// }

	// err = InstallMonitoringDependencies(ctx, kubeProvider)
	// if err != nil {
	// 	return err, nil
	// }

	/*
		err = InstallLinkerD(ctx)
		if err != nil {
			return err
		} */

	// NGINX Ingress Operator

	// External DNS

	// Prometheus

	/*
		err = InstallKafka(ctx)
		if err != nil {
			return err
		}
	*/

	return nil, SecretsProvider
}
