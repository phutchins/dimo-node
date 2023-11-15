package infrastructure

import (
	"github.com/dimo/dimo-node/utils"
	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const osImage = "debian-11"
const sshUser = "pulumi"
const pubKeyPath = "./infrastructure/keys/pulumi_key.pub"
const privKeyPath = "./infrastructure/keys/pulumi_key"

var firewallOpenPorts = []string{
	"22",    // SSH
	"6443",  // Kubernetes API
	"31544", // Postgres Operator UI
}

// Configure what type of deployment and where it should be deployed
const cloudProvider = "gcp"
const deploymentType = "gke"
const projectName = "dimo-dev"

// Specific configuration that will likely end up being dynamic
const instanceTag = pulumi.String("dimo")

// Specific configuration for GCP
// const zone = pulumi.String("us-central1-a")
// const machineType = "f1-micro"
const region = "us-central1"

// Configure your own access to the cluster or VM
const whitelistIp = pulumi.String("24.30.56.126/32")

// Define variables needed outside the BuildInfrastructure() function
var KubeConfig *pulumi.StringOutput
var KubeProvider *kubernetes.Provider

func BuildInfrastructure(ctx *pulumi.Context) (*kubernetes.Provider, error) {
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
		return nil, err
	}

	network, subnetwork, err := CreateNetwork(ctx, cloudProvider, projectName, region)

	switch deploymentType {
	case "k3s":
		inst, err := CreateK3sCluster(ctx, network, subnetwork)
		if err != nil {
			return nil, err
		}

		accessConfigs := inst.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs()
		publicIp := accessConfigs.Index(pulumi.Int(0)).NatIp().Elem()

		connection, err := GetKubeHostConnection(ctx, publicIp, sshPrivKey)
		if err != nil {
			return nil, err
		}

		KubeConfig, err = GetKubeConfigForK3s(ctx, connection, inst)
		if err != nil {
			return nil, err
		}

		k3sProvider, err := NewKubeProviderForK3s(ctx, KubeConfig)
		if err != nil {
			return nil, err
		}

		// Note that .Elem() essentially dereferences the output pointer to give us an unwrapped value we can use
		internalIp := inst.NetworkInterfaces.Index(pulumi.Int(0)).NetworkIp().Elem()

		KubeProvider = k3sProvider

		ctx.Export("instanceName", inst.Name)
		ctx.Export("publicIp", publicIp)
		ctx.Export("internalIp", internalIp)
	case "gke":
		cluster, err := CreateGKECluster(
			ctx,
			projectName,
			region,
		)
		if err != nil {
			return nil, err
		}

		// Create the Kubernetes provider
		k8sProvider, err := NewKubernetesProvider(ctx, cluster)
		if err != nil {
			return nil, err
		}

		ctx.Export("k8sProvider", k8sProvider.URN())
		KubeProvider = k8sProvider
	case "aks":
		//inst, err = CreateAKSCluster(ctx, network, subnetwork)
	default:
		return nil, err
	}

	err = utils.CreateNamespaces(ctx, KubeProvider, []string{"dimo"})
	if err != nil {
		return nil, err
	}

	return KubeProvider, nil
}
