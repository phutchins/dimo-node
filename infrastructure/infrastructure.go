package infrastructure

import (
	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
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
const whitelistIp = pulumi.String("24.30.56.126/32")
const pubKeyPath = "./infrastructure/keys/pulumi_key.pub"
const privKeyPath = "./infrastructure/keys/pulumi_key"

var firewallOpenPorts = []string{"22", "6443", "31544"}

// Define variables needed outside the BuildInfrastructure() function
var KubeConfig *pulumi.StringOutput
var KubeProvider *kubernetes.Provider

func BuildInfrastructure(ctx *pulumi.Context) (err error) {
	// Read the SSH Keys from Disk
	sshKey, sshPrivKey, err := ReadSSHKeysFromDisk(pubKeyPath, privKeyPath)
	if err != nil {
		panic(err)
	}

	// Create a GCP network

	_, err = compute.NewProjectMetadata(ctx, "ssh-keys", &compute.ProjectMetadataArgs{
		Metadata: pulumi.StringMap{
			"ssh-keys": pulumi.Sprintf("%s:%s", sshUser, sshKey),
		},
	})
	if err != nil {
		return err
	}

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
				Ports:    utils.ToPulumiStringArray(firewallOpenPorts),
			},
		},
		Direction: pulumi.String("INGRESS"),
		SourceRanges: pulumi.StringArray{
			whitelistIp,
		},
		TargetTags: pulumi.StringArray{
			pulumi.String(instanceTag),
		},
	}, pulumi.DependsOn([]pulumi.Resource{subnetwork}))
	if err != nil {
		return err
	}

	inst, err := CreateK3sCluster(ctx, network, subnetwork)
	if err != nil {
		return err
	}

	// Note that .Elem() essentially dereferences the output pointer to give us an unwrapped value we can use
	internalIp := inst.NetworkInterfaces.Index(pulumi.Int(0)).NetworkIp().Elem()
	accessConfigs := inst.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs()
	publicIp := accessConfigs.Index(pulumi.Int(0)).NatIp().Elem()

	ctx.Export("instanceName", inst.Name)
	ctx.Export("publicIp", publicIp)
	ctx.Export("internalIp", internalIp)
	ctx.Export("subnetworkName", subnetwork.Name)
	ctx.Export("firewallName", firewall.Name)

	connection, err := GetKubeHostConnection(ctx, publicIp, sshPrivKey)
	if err != nil {
		return err
	}

	KubeConfig, err = GetKubeConfigForK3s(ctx, connection, inst)
	if err != nil {
		return err
	}

	KubeProvider, err = NewKubeProviderForK3s(ctx, KubeConfig)
	if err != nil {
		return err
	}

	err = CreateNamespaces(ctx, KubeProvider, []string{"dimo"})
	if err != nil {
		return err
	}

	return nil
}
