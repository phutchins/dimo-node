package infrastructure

import (
	"strings"

	"github.com/dimo/dimo-node/utils"
	//"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/container"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
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
// K3s (GCP)
// const cloudProvider = "gcp"
// const deploymentType = "k3s"
// GCP (GCP)
// const cloudProvider = "gcp"
// const deploymentType = "gke"
// EKS (AWS)
//const cloudProvider = "aws"
//const deploymentType = "eks"

//const projectName = "dimo-dev-401815"
//const createNodePools = false // Disable to save costs

// Specific configuration that will likely end up being dynamic
const instanceTag = pulumi.String("dimo")

// Specific configuration for GCP
// const zone = pulumi.String("us-central1-a")
// const machineType = "f1-micro"
//const region = "us-central1"

// Configure your own access to the cluster or VM
//const whitelistIp = pulumi.String("24.30.56.126/32")

// Define variables needed outside the BuildInfrastructure() function
var KubeConfig *pulumi.StringOutput
var KubeProvider *kubernetes.Provider
var Network *compute.Network
var Subnetwork *compute.Subnetwork
var Cluster *container.Cluster

func BuildInfrastructure(ctx *pulumi.Context) (*kubernetes.Provider, error) {
	conf := config.New(ctx, "")
	cloudProvider := conf.Require("cloud-provider")
	deploymentType := conf.Require("deployment-type")
	// TODO: Set sane defaults for these if not set
	projectName := conf.Get("project-name")
	createNodePools := conf.GetBool("create-node-pools")
	region := conf.Get("region")
	location := conf.Get("location")
	locationsStr := conf.Get("locations")
	locations := strings.Split(locationsStr, ",")
	whitelistIp := conf.Get("whitelist-ip")

	network, subnetwork, err := CreateNetwork(ctx, cloudProvider, region, projectName, whitelistIp)
	if err != nil {
		return nil, err
	}

	Network = network
	Subnetwork = subnetwork

	// TODO: Move this switch statement into its own file and call it with GetKubeProvider()
	switch deploymentType {
	case "k3s":
		// Read the SSH Keys from Disk
		pubKey, privKey, err := ReadSSHKeysFromDisk(pubKeyPath, privKeyPath)
		if err != nil {
			panic(err)
		}

		err = AddSSHKeysMetadata(ctx, cloudProvider, pubKey, privKey)
		if err != nil {
			return nil, err
		}

		inst, err := CreateK3sCluster(ctx, network, subnetwork)
		if err != nil {
			return nil, err
		}

		accessConfigs := inst.NetworkInterfaces.Index(pulumi.Int(0)).AccessConfigs()
		publicIp := accessConfigs.Index(pulumi.Int(0)).NatIp().Elem()

		connection, err := GetKubeHostConnection(ctx, publicIp, privKey)
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
			location,
		)
		if err != nil {
			return nil, err
		}

		if createNodePools {
			err = CreateGKENodePools(ctx, projectName, cluster, region, locations)
			if err != nil {
				return nil, err
			}
		}

		// Create the Kubernetes provider
		k8sProvider, err := NewGKEKubernetesProvider(ctx, cluster)
		if err != nil {
			return nil, err
		}

		err = CreateGKEKubePriorities(ctx, cluster, k8sProvider)
		if err != nil {
			return nil, err
		}

		KubeProvider = k8sProvider
	case "eks":
		cluster, err := CreateEKSKubernetesCluster(
			ctx,
			projectName,
			region,
		)
		if err != nil {
			return nil, err
		}

		if createNodePools {
			err = CreateEKSKubernetesNodePools(ctx, projectName, cluster, region)
			if err != nil {
				return nil, err
			}
		}

		// Create the Kubernetes provider
		k8sProvider, err := NewEKSKubernetesProvider(ctx, cluster)
		if err != nil {
			return nil, err
		}

		KubeProvider = k8sProvider
	default:
		return nil, err
	}

	ctx.Export("k8sProvider", KubeProvider.URN())

	_, err = utils.CreateNamespaces(ctx, KubeProvider, []string{"dimo"})
	if err != nil {
		return nil, err
	}

	return KubeProvider, nil
}
