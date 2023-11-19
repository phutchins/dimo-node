package infrastructure

import (
	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-command/sdk/go/command/remote"

	//"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateK3sCluster(
	ctx *pulumi.Context,
	network *compute.Network,
	subnetwork *compute.Subnetwork) (
	*compute.Instance,
	error) {
	instanceTags := utils.ToPulumiStringArray([]string{"dimo"})

	const metadataStartupScript = `#!/bin/bash sudo apt-get update && sudo apt install -y jq`

	// Reserve a new public IP
	publicAddress, err := compute.NewAddress(ctx, "publicip1", nil)
	if err != nil {
		return nil, err
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
	}, pulumi.DependsOn([]pulumi.Resource{publicAddress}))
	if err != nil {
		return nil, err
	}

	return inst, nil
}

func GetKubeHostConnection(
	ctx *pulumi.Context,
	publicIp pulumi.StringOutput,
	sshPrivKey string,
) (remote.ConnectionArgs, error) {
	// Create remote connection
	connection := remote.ConnectionArgs{
		Host:       publicIp,
		PrivateKey: pulumi.String(string(sshPrivKey)),
		User:       pulumi.String("pulumi"),
	}

	return connection, nil
}

func GetKubeConfigForK3s(
	ctx *pulumi.Context,
	connection remote.ConnectionArgs,
	inst *compute.Instance) (*pulumi.StringOutput, error) {
	internalIp := inst.NetworkInterfaces.Index(pulumi.Int(0)).NetworkIp().Elem()
	accessConfigs := inst.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs()
	publicIp := accessConfigs.Index(pulumi.Int(0)).NatIp().Elem()

	k3sCmdString := pulumi.Sprintf("curl -sfL https://get.k3s.io | sh -s -- --bind-address %s --tls-san %s --advertise-address %s --advertise-address %s --disable servicelb --write-kubeconfig-mode=644", internalIp, publicIp, internalIp, internalIp)

	_, err := remote.NewCommand(ctx, "k3sinstall", &remote.CommandArgs{
		Create:     k3sCmdString,
		Connection: connection,
	}, pulumi.DependsOn([]pulumi.Resource{inst}))
	if err != nil {
		return nil, err
	}

	getKubeConfigCmd := pulumi.Sprintf("sudo cat /etc/rancher/k3s/k3s.yaml | sed \"s/.*server: .*/    server: https:\\/\\/%s:6443/g\"", publicIp)

	getKubeConfig, err := remote.NewCommand(ctx, "getkubeconfig", &remote.CommandArgs{
		Create:     getKubeConfigCmd,
		Update:     getKubeConfigCmd,
		Connection: connection,
	}, pulumi.DependsOn([]pulumi.Resource{inst}))
	if err != nil {
		return nil, err
	}

	kubeConfig := getKubeConfig.Stdout.ApplyT(func(s string) string {
		return s
	}).(pulumi.StringOutput)

	return &kubeConfig, nil
}

func NewKubeProviderForK3s(ctx *pulumi.Context, KubeConfig *pulumi.StringOutput) (*kubernetes.Provider, error) {
	// Create the Kubernetes provider
	kubeProvider, err := kubernetes.NewProvider(ctx, "k3s", &kubernetes.ProviderArgs{
		Kubeconfig: KubeConfig,
	})
	if err != nil {
		return nil, err
	}

	return kubeProvider, nil
}

/* implement this later if necessary
// Create the metallb-system namespace
metallbNamespace, err := corev1.NewNamespace(ctx, "metallb", &corev1.NamespaceArgs{
	Metadata: &metav1.ObjectMetaArgs{
		Name: pulumi.String("metallb-system"),
	},
}, pulumi.Provider(kubeProvider))
if err != nil {
	return err
}

// Deploy metallb with helm
_, err = helm.NewChart(ctx, "metallb", helm.ChartArgs{
	Chart: pulumi.String("metallb"),
	FetchArgs: helm.FetchArgs{
		Repo: pulumi.String("https://metallb.github.io/metallb"),
	},
	Namespace: metallbNamespace.Metadata.Name().Elem(),
	Values:    pulumi.Map{},
}, pulumi.Provider(kubeProvider),
	pulumi.DependsOn([]pulumi.Resource{metallbNamespace}))
if err != nil {
	return err
}
*/
