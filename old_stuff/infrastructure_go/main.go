package main

import (
	"github.com/pulumi/pulumi-google-native/sdk/go/google/compute/beta",
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Define the VM startup script that will install k3s
		startupScript := `#!/bin/bash
sudo apt-get update
curl -sfL https://get.k3s.io | sh -`

		// Create a new GCP VM Instance
		vm, err := beta.NewInstance(ctx, "myVm", &beta.InstanceArgs{
			Zone:                       pulumi.String("us-central1-a"),
			Name:                       pulumi.String("my-instance"),
			MachineType:                pulumi.String("e2-medium"), // or choose another instance type
			MinCpuPlatform:             pulumi.String("Intel Haswell"), // or choose another CPU platform
			BootDisk: &beta.InstanceBootDiskArgs{
				InitializeParams: &beta.InstanceBootDiskInitializeParamsArgs{
					Image: pulumi.String("projects/debian-cloud/global/images/debian-10-buster-v20210609"), // or choose another image
				},
			},
			NetworkInterfaces: beta.InstanceNetworkInterfaceArray{
				&beta.InstanceNetworkInterfaceArgs{
					AccessConfigs: beta.InstanceAccessConfigArray{
						&beta.InstanceAccessConfigArgs{},
					},
				},
			},
			MetadataStartupScript: pulumi.String(startupScript),
		})
		if err != nil {
			return err
		}

		// Export the VM instance name
		ctx.Export("instanceName", vm.Name)
		return nil
	})
}