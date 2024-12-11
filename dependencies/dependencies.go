package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Define variables needed globally in the dependencies package

func InstallDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error, SecretsProvider *helm.Chart) {
	err = InstallLetsEncrypt(ctx, kubeProvider)
	if err != nil {
		return err, nil
	}

	err = InstallDatabaseDependencies(ctx)
	if err != nil {
		return err, nil
	}

	err = InstallCertificateDependencies(ctx, kubeProvider)
	if err != nil {
		return err, nil
	}

	err, SecretsProvider = InstallSecretsDependencies(ctx, kubeProvider)
	if err != nil {
		return err, nil
	}

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
