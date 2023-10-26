package infrastructure

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi-command/sdk/v3/go/command/remote"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Machine type
const machineType = "f1-micro"
const osImage = "debian-11"
const instanceTag = pulumi.String("dimo")
const zone = pulumi.String("us-central1-a")
const region = pulumi.String("us-central1")
const kubePort = pulumi.Int(6443)

func ConvertString(s *string) pulumi.String {
	return pulumi.String(fmt.Sprintf("%v", s))
}

func toPulumiStringArray(a []string) pulumi.StringArrayInput {
	var res []pulumi.StringInput
	for _, s := range a {
		res = append(res, pulumi.String(s))
	}
	return pulumi.StringArray(res)
}

func Run() {
	// Get public key from file and create a GCP metadata item
	pubKey, err := os.ReadFile("./keys/pulumi_key.pub")
	if err != nil {
		panic(err)
	}
	sshKeys := pulumi.String(string(pubKey))

	privKey, err := os.ReadFile("./keys/pulumi_key")
	if err != nil {
		panic(err)
	}
	sshPrivKey := pulumi.String(string(privKey))

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
			Region:      pulumi.String(region),
		}, pulumi.DependsOn([]pulumi.Resource{network}))
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
		}, pulumi.DependsOn([]pulumi.Resource{subnetwork}))

		/*
			instanceLabels := pulumi.Map(map[string]interface{}{
				"app": "dimo",
			})

			m := pulumi.NewMapInput("m", map[string]string{
				"key1": "value1",
				"key2": "value2",
			})
		*/

		/*
			instanceLabels := pulumi.Map{
				"app": instanceTag,
			}
		*/

		instanceTags := toPulumiStringArray([]string{"dimo"})

		const metadataStartupScript = `#!/bin/bash sudo apt-get update && sudo apt install -y jq`

		// Reserve a new public IP
		publicAddress, err := compute.NewAddress(ctx, "publicip1", nil)
		if err != nil {
			return err
		}

		inst, err := compute.NewInstance(ctx, "instance", &compute.InstanceArgs{
			BootDisk: &compute.InstanceBootDiskArgs{
				InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
					Image: pulumi.String(osImage),
				},
			},
			MachineType:           pulumi.String("n1-standard-1"),
			Tags:                  instanceTags,
			MetadataStartupScript: pulumi.String(metadataStartupScript),
			NetworkInterfaces: &compute.InstanceNetworkInterfaceArray{
				&compute.InstanceNetworkInterfaceArgs{
					Network: network.ID(),
					AccessConfigs: compute.InstanceNetworkInterfaceAccessConfigArray{
						&compute.InstanceNetworkInterfaceAccessConfigArgs{
							NatIp: publicAddress.Address, // Can use NatIp: pulumi.String("") to get an ephemeral IP
						},
					},
					Subnetwork: subnetwork.ID(),
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{subnetwork, firewall, publicAddress}))
		if err != nil {
			return err
		}

		internalIP := inst.NetworkInterfaces.Index(pulumi.Int(0)).NetworkIp()
		publicIP := publicAddress.Address
		/*
			publicIP := inst.NetworkInterfaces.ApplyT(
				func(ni []compute.InstanceNetworkInterface) *string {
					return ni[0].AccessConfigs[0].NatIp
				})
		*/

		ctx.Export("instanceName", inst.Name)
		ctx.Export("subnetworkName", subnetwork.Name)
		ctx.Export("firewallName", firewall.Name)
		ctx.Export("publicIp", publicIP)
		ctx.Export("internalIp", internalIP)

		// Create remote connection
		connection := remote.ConnectionArgs{
			Host:       publicAddress.Address,
			PrivateKey: pulumi.String(string(sshPrivKey)),
			User:       pulumi.String("pulumi"),
		}

		k3sCmdString := pulumi.All(publicIP, kubePort, internalIP).ApplyT(
			func(args []interface{}) string {


		k3sCmd, err := remote.NewCommand(ctx, "k3sinstall", &remote.CommandArgs{
			Create:     pulumi.String("curl -sfL https://get.k3s.io | sh -"),
			Connection: connection,
		})

		ctx.Export("k3sCmd", k3sCmd)

		return nil
	})
}
