package infrastructure

import (
	"os"

	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Machine type
const machineType = "f1-micro"
const osImage = "debian-11"
const instanceTag = "dimo-node"

func Run() {
	// Get public key from file and create a GCP metadata item
	pubKey, err := os.ReadFile("./keys/id_rsa.pub")
	if err != nil {
		panic(err)
	}
	sshKeys := pulumi.String(string(pubKey))

	// Create a GCP network
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := compute.NewProjectMetadata(ctx, "ssh-keys", &compute.ProjectMetadataArgs{
			Metadata: pulumi.StringMap{
				"ssh-keys": pulumi.String(sshKeys),
			},
		})

		network, err := compute.NewNetwork(ctx, "network", &compute.NetworkArgs{
			AutoCreateSubnetworks: pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		// Create a GCP Subnetwork
		subnetwork, err := compute.NewSubnetwork(ctx, "subnetwork", &compute.SubnetworkArgs{
			IpCidrRange: pulumi.String("10.0.1.0/24"),
			Network:     network.Name,
		})
		if err != nil {
			return err
		}

		// Create firewall rule to allow appropriate traffic in
		firewall, err := compute.NewFirewall(ctx, "firewall", &compute.FirewallArgs{
			Network: network.Name,
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports: pulumi.StringArray{
						pulumi.String("22"),
						pulumi.String("31544"),
					},
				},
			},
			Direction: pulumi.String("INGRESS"),
			SourceRanges: pulumi.StringArray{
				pulumi.String("24.30.56.126/32"),
			},
			TargetTags: pulumi.StringArray{
				pulumi.String(instanceTag),
			},
		})

		inst, err := compute.NewInstance(ctx, "instance", &compute.InstanceArgs{
			BootDisk: &compute.InstanceBootDiskArgs{
				InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
					Image: pulumi.String(osImage),
				},
			},
			MachineType: pulumi.String("n1-standard-1"),
			NetworkInterfaces: &compute.InstanceNetworkInterfaceArray{
				&compute.InstanceNetworkInterfaceArgs{
					Network: network.Name,
				},
			},
		})
		if err != nil {
			return err
		}

		ctx.Export("instanceName", inst.Name)
		ctx.Export("subnetworkName", subnetwork.Name)
		ctx.Export("firewallName", firewall.Name)

		return nil
	})
}
