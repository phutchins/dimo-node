package main

import (
	"github.com/dimo/dimo-node/applications"
	"github.com/dimo/dimo-node/dependencies"
	"github.com/dimo/dimo-node/infrastructure"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		kubeProvider, err := infrastructure.BuildInfrastructure(ctx)
		if err != nil {
			return err
		}

		dependencies.InstallDependencies(ctx, kubeProvider)

		err = applications.InstallApplications(ctx, kubeProvider)
		if err != nil {
			return err
		}

		return nil
	})
}
