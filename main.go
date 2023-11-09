package main

import (
	"github.com/dimo/dimo-node/dependencies"
	"github.com/dimo/dimo-node/infrastructure"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		infrastructure.BuildInfrastructure(ctx)

		dependencies.InstallDependencies(ctx, infrastructure.KubeConfig)

		return nil
	})
}
