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
const sshUser = "pulumi"

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
	sshKey := pulumi.String(string(pubKey))

	privKey, err := os.ReadFile("./keys/pulumi_key")
	if err != nil {
		panic(err)
	}
	sshPrivKey := pulumi.String(string(privKey))

	// Create a GCP network
	pulumi.Run(func(ctx *pulumi.Context) error {
		_, err := compute.NewProjectMetadata(ctx, "ssh-keys", &compute.ProjectMetadataArgs{
			Metadata: pulumi.StringMap{
				"ssh-keys": pulumi.Sprintf("%s:%s", sshUser, sshKey),
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

		// Note that .Elem() essentially dereferences the output pointer to give us an unwrapped value we can use
		internalIp := inst.NetworkInterfaces.Index(pulumi.Int(0)).NetworkIp().Elem()
		accessConfigs := inst.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs()
		publicIp := accessConfigs.Index(pulumi.Int(0)).NatIp().Elem()

		/*
			publicIp := pulumi.All(inst.NetworkInterfaces).ApplyT(
				func(ni []compute.InstanceNetworkInterface) pulumi.StringOutput {
					ip := ni[0].AccessConfigs[0].NatIp
					return pulumi.Sprintf("%s", ip)
				})
		*/

		//publicIp := pulumi.String("23.23.23.23")
		//publicIp := inst.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs().Index(pulumi.Int(0)).NatIp()
		//publicIp := publicAddress.Address

		/*publicIp := inst.NetworkInterfaces.ApplyT(
		func(ni []compute.InstanceNetworkInterface) *string {
			return ni[0].AccessConfigs[0].NatIp
		})
		*/

		ctx.Export("instanceName", inst.Name)
		ctx.Export("subnetworkName", subnetwork.Name)
		ctx.Export("firewallName", firewall.Name)
		ctx.Export("publicIp", publicIp)
		ctx.Export("internalIp", internalIp)

		// Create remote connection
		connection := remote.ConnectionArgs{
			Host:       publicIp,
			PrivateKey: pulumi.String(string(sshPrivKey)),
			User:       pulumi.String("pulumi"),
		}

		k3sCmdString := pulumi.Sprintf("curl -sfL https://get.k3s.io | sh -s -- --bind-address %s --tls-san %s --advertise-address %s --advertise-address %s --disable servicelb --write-kubeconfig-mode=644", internalIp, publicIp, internalIp, internalIp)

		_, err = remote.NewCommand(ctx, "k3sinstall", &remote.CommandArgs{
			Create:     k3sCmdString,
			Connection: connection,
		})
		if err != nil {
			return err
		}

		getKubeConfigCmd := pulumi.Sprintf("sudo cat /etc/rancher/k3s/k3s.yaml | sed 's/.*server: .*/    server: https:\\/\\/%s:6443/g'", publicIp)

		getKubeConfig, err := remote.NewCommand(ctx, "getkubeconfig", &remote.CommandArgs{
			Create:     getKubeConfigCmd,
			Connection: connection,
		})
		if err != nil {
			return err
		}

		kubeConfig := getKubeConfig.Stdout.ApplyT(func(s string) string {
			return s
		})

		ctx.Export("kubeConfig", kubeConfig)

		return nil
	})
}
