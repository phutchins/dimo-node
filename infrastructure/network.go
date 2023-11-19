package infrastructure

import (
	"fmt"

	"github.com/dimo/dimo-node/utils"
	//"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateNetwork(ctx *pulumi.Context, cloudProvider string, region string, projectName string) (*compute.Network, *compute.Subnetwork, error) {
	switch cloudProvider {
	case "aws":
		return CreateAWSNetwork(ctx, region, projectName)
	case "gcp":
		return CreateGCPNetwork(ctx, region, projectName)
	default:
		return nil, nil, fmt.Errorf("cloud provider %s not supported", cloudProvider)
	}
}

func CreateGCPNetwork(ctx *pulumi.Context, region string, projectName string) (*compute.Network, *compute.Subnetwork, error) {
	networkName := fmt.Sprintf("%s-network", projectName)
	subnetworkName := fmt.Sprintf("%s-subnetwork", projectName)
	firewallName := fmt.Sprintf("%s-firewall", projectName)

	network, err := compute.NewNetwork(ctx, networkName, &compute.NetworkArgs{
		AutoCreateSubnetworks: pulumi.Bool(false),
	})
	if err != nil {
		return nil, nil, err
	}

	// Create a GCP Subnetwork
	subnetwork, err := compute.NewSubnetwork(ctx, subnetworkName, &compute.SubnetworkArgs{
		IpCidrRange: pulumi.String("10.0.1.0/24"),
		Network:     network.Name,
		Region:      pulumi.String("us-central1"),
		//Region:      pulumi.String(region),
	}, pulumi.DependsOn([]pulumi.Resource{network}))
	if err != nil {
		return nil, nil, err
	}

	// Create firewall rule to allow appropriate traffic in
	firewall, err := compute.NewFirewall(ctx, firewallName, &compute.FirewallArgs{
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
		return nil, nil, err
	}

	ctx.Export("subnetworkName", subnetwork.Name)
	ctx.Export("firewallName", firewall.Name)

	return network, subnetwork, nil
}

func CreateAWSNetwork(ctx *pulumi.Context, region string, projectName string) (*compute.Network, *compute.Subnetwork, error) {
	return nil, nil, nil
}
