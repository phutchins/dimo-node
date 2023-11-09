package infrastructure

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi-command/sdk/go/command/remote"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
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

var firewallOpenPorts = []string{"22", "6443", "31544"}

// Define variables needed outside the BuildInfrastructure() function
var KubeConfig pulumi.StringOutput

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

func BuildInfrastructure(ctx *pulumi.Context) (err error) {
	// Get public key from file and create a GCP metadata item
	pubKey, err := os.ReadFile("./infrastructure/keys/pulumi_key.pub")
	if err != nil {
		panic(err)
	}
	sshKey := pulumi.String(string(pubKey))

	privKey, err := os.ReadFile("./infrastructure/keys/pulumi_key")
	if err != nil {
		panic(err)
	}
	sshPrivKey := pulumi.String(string(privKey))

	// Create a GCP network

	// Configure the stack name
	//ctx.SetStack(stackName)

	_, err = compute.NewProjectMetadata(ctx, "ssh-keys", &compute.ProjectMetadataArgs{
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
				Ports:    toPulumiStringArray(firewallOpenPorts),
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
	}, pulumi.DependsOn([]pulumi.Resource{inst}))
	if err != nil {
		return err
	}

	testCommand := pulumi.Sprintf("uname -a")
	runTestCommand, err := remote.NewCommand(ctx, "test", &remote.CommandArgs{
		Create:     testCommand,
		Connection: connection,
	}, pulumi.DependsOn([]pulumi.Resource{inst, firewall}))
	if err != nil {
		return err
	}

	runTestCommand.Stdout.ApplyT(func(s string) error {
		fmt.Printf("Test: %s", s)
		return nil
	})

	getKubeConfigCmd := pulumi.Sprintf("sudo cat /etc/rancher/k3s/k3s.yaml | sed \"s/.*server: .*/    server: https:\\/\\/%s:6443/g\"", publicIp)

	getKubeConfig, err := remote.NewCommand(ctx, "getkubeconfig", &remote.CommandArgs{
		Create:     getKubeConfigCmd,
		Connection: connection,
	}, pulumi.DependsOn([]pulumi.Resource{inst}))
	if err != nil {
		return err
	}

	/*
		kubeConfig := getKubeConfig.Stdout.ApplyT(func(s) string {
			return string(s)
		})(pulumi.StringOutput) */

	//ctx.Export("kubeConfig", pulumi.String(kubeConfig))

	/* Implement this when we start using GKE clusters
	// Reference: https://www.pulumi.com/registry/packages/kubernetes/how-to-guides/gke/
	// Manufacture a GKE-style kubeconfig. Note that this is slightly "different"
	// because of the way GKE requires gcloud to be in the picture for cluster
	// authentication (rather than using the client cert/key directly).
	export const kubeconfig = pulumi.
	all([ cluster.name, cluster.endpoint, cluster.masterAuth ]).
	apply(([ name, endpoint, masterAuth ]) => {
			const context = `${gcp.config.project}_${gcp.config.zone}_${name}`;
			return `apiVersion: v1
	clusters:
	- cluster:
	certificate-authority-data: ${masterAuth.clusterCaCertificate}
	server: https://${endpoint}
	name: ${context}
	contexts:
	- context:
	cluster: ${context}
	user: ${context}
	name: ${context}
	current-context: ${context}
	kind: Config
	preferences: {}
	users:
	- name: ${context}
	user:
	exec:
		apiVersion: client.authentication.k8s.io/v1beta1
		command: gke-gcloud-auth-plugin
		installHint: Install gke-gcloud-auth-plugin for use with kubectl by following
			https://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke
		provideClusterInfo: true
	`;
	});

	// Create a Kubernetes provider instance that uses our cluster from above.
	const clusterProvider = new k8s.Provider(name, {
	kubeconfig: kubeconfig,
	});
	*/

	/* for troubleshooting
	getKubeConfig.Stdout.ApplyT(func(s string) error {
		fmt.Printf("Kubeconfig: %s", s)
		return nil
	}) */

	KubeConfig = getKubeConfig.Stdout.ApplyT(func(s string) string {
		return s
	}).(pulumi.StringOutput)

	ctx.Export("kubeConfig", getKubeConfig.Stdout.ApplyT(func(s string) string {
		return s
	}).(pulumi.StringOutput))

	ctx.Export("getKubeConfig", getKubeConfig)

	kubeProvider, err := kubernetes.NewProvider(ctx, "k3s", &kubernetes.ProviderArgs{
		Kubeconfig: getKubeConfig.Stdout.ApplyT(func(s string) string {
			return s
		}).(pulumi.StringOutput),
	}, pulumi.DependsOn([]pulumi.Resource{inst, firewall, getKubeConfig}))
	if err != nil {
		return err
	}

	// Create a namespace
	_, err = corev1.NewNamespace(ctx, "dimo", &corev1.NamespaceArgs{
		Metadata: &metav1.ObjectMetaArgs{
			Name: pulumi.String("dimo"),
		},
	}, pulumi.Provider(kubeProvider),
		pulumi.DependsOn([]pulumi.Resource{inst, kubeProvider}))
	if err != nil {
		return err
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

	return nil
}
