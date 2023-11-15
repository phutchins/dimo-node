package dependencies

import (
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
)

// Define variables needed globally in the dependencies package

func InstallDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	err = InstallDatabaseDependencies(ctx)
	if err != nil {
		return err
	}

	err = InstallLinkerD(ctx)
	if err != nil {
		return err
	}

	err = InstallKafka(ctx)
	if err != nil {
		return err
	}

	return nil
}
