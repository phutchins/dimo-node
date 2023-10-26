package infrastructure

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Machine type
const machineType = "f1-micro"
const osImage = "debian-cloud/debian-11"
const instanceTag = "dimo-node"
const zone = "us-central1-a"
const region = "us-central1"

func Run() {
	// Get public key from file and create a GCP metadata item
	pubKey, err := os.ReadFile("./keys/pulumi_key.pub")
	if err != nil {
		panic(err)
	}
	sshKeys := pulumi.String(string(pubKey))

	// Create a GCP network
	pulumi.Run(func(ctx *pulumi.Context) error {
		fmt.Printf("HI\n")
		fmt.Printf("HI\n")
		_, err := compute.NewProjectMetadata(ctx, "ssh-keys", &compute.ProjectMetadataArgs{
			Metadata: pulumi.StringMap{
				"ssh-keys": pulumi.String(sshKeys),
			},
		})

		network, err := compute.NewNetwork(ctx, "network-1", &compute.NetworkArgs{
			AutoCreateSubnetworks: pulumi.Bool(false),
		})
		if err != nil {
			return err
		}

		// Create a GCP Subnetwork
		subnetwork, err := compute.NewSubnetwork(ctx, "subnetwork-1", &compute.SubnetworkArgs{
			IpCidrRange: pulumi.String("10.0.1.0/24"),
			Network:     network.ID(),
			Region:      pulumi.String("us-central1"),
		}, pulumi.DependsOn([]pulumi.Resource{network}))

		if err != nil {
			return err
		}

		// Create firewall rule to allow appropriate traffic in
		firewall, err := compute.NewFirewall(ctx, "firewall-1", &compute.FirewallArgs{
			Network:    network.ID(),
			Subnetwork: subnetwork.ID(),
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
		}, pulumi.DependsOn([]pulumi.Resource{subnetwork}))

		instanceLabels := pulumi.StringMap{
			"app": pulumi.String("dimo"),
		}

		fmt.Printf("HI")

		/*
			// Create a GCP Instance
			inst, err := compute.NewInstance(ctx, "instance", &compute.InstanceArgs{
				Zone: pulumi.String(zone),
				BootDisk: &compute.InstanceBootDiskArgs{
					InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
						Image:  pulumi.String(osImage),
						Labels: instanceLabels, // <- This shows an error in IDE but seems to work fine
					},
				},
				MachineType: pulumi.String(machineType),
				NetworkInterfaces: &compute.InstanceNetworkInterfaceArray{
					&compute.InstanceNetworkInterfaceArgs{
						//Subnetwork: subnetwork.SelfLink.ToStringOutput(),
						Subnetwork: pulumi.String("default"),
						Network:    pulumi.String("default"),
					},
				},
			}, pulumi.DependsOn([]pulumi.Resource{subnetwork, firewall}))
			if err != nil {
				return err
			}
		*/

		/* ctx.Export("instanceName", inst.Name)*/
		ctx.Export("networkName", network.Name)
		ctx.Export("subnetworkName", subnetwork.Name)
		ctx.Export("firewallName", firewall.Name)

		return nil
	})
}
