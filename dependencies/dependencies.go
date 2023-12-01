package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Define variables needed globally in the dependencies package

func InstallDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	err = InstallDatabaseDependencies(ctx)
	if err != nil {
		return err
	}

	err = InstallCertificateDependencies(ctx, kubeProvider)
	if err != nil {
		return err
	}

	err = InstallSecretsDependencies(ctx, kubeProvider)
	if err != nil {
		return err
	}

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

	return nil
}