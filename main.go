package main

import (
	"log"

	"github.com/dimo/dimo-node/applications"
	"github.com/dimo/dimo-node/dependencies"
	"github.com/dimo/dimo-node/infrastructure"
	"github.com/dimo/dimo-node/utils"
	"github.com/joho/godotenv"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := utils.EnsureGKEAuth(); err != nil {
		log.Fatal("Error setting up GKE auth:", err)
	}

	pulumi.Run(func(ctx *pulumi.Context) error {
		kubeProvider, err := infrastructure.BuildInfrastructure(ctx)
		if err != nil {
			return err
		}

		err, SecretsProvider := dependencies.InstallDependencies(ctx, kubeProvider)

		err = applications.InstallApplications(ctx, kubeProvider, SecretsProvider)
		if err != nil {
			return err
		}

		return nil
	})
}
