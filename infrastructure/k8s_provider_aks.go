package infrastructure

/*
import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/containerservice/mgmt/containerservice"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)



func createAKSKubernetesCluster(ctx context.Context, plumiContext *schema.ResourceData) (*containerservice.ManagedCluster, error) {
	// Get AKS credentials
	aksCredentials, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to get AKS credentials: %v", err)
	}

	// Create AKS client
	aksClient := containerservice.NewManagedClustersClient(plumiContext.Get("subscription_id").(string))
	aksClient.Authorizer = aksCredentials

	// Define cluster properties
	clusterName := plumiContext.Get("cluster_name").(string)
	resourceGroupName := plumiContext.Get("resource_group_name").(string)
	location := plumiContext.Get("location").(string)

	// Define node pool properties
	mediumNodePoolName := "medium"
	mediumNodePoolVMSize := "Standard_D4_v3"
	mediumNodePoolCount := int32(1)
	mediumNodePoolOSDiskSizeGB := int32(30)
	mediumNodePoolVNetSubnetID := plumiContext.Get("vnet_subnet_id").(string)

	smallNodePoolName := "small"
	smallNodePoolVMSize := "Standard_D2_v3"
	smallNodePoolCount := int32(1)
	smallNodePoolOSDiskSizeGB := int32(30)
	smallNodePoolVNetSubnetID := plumiContext.Get("vnet_subnet_id").(string)

	// Define cluster properties
	clusterProperties := &containerservice.ManagedClusterProperties{
		DNSPrefix:         &clusterName,
		NodeResourceGroup: &resourceGroupName,
		KubernetesVersion: to.StringPtr("1.19.11"),
		AgentPoolProfiles: &[]containerservice.ManagedClusterAgentPoolProfile{
			{
				Name:         &mediumNodePoolName,
				Count:        &mediumNodePoolCount,
				VMSize:       &mediumNodePoolVMSize,
				OsDiskSizeGB: &mediumNodePoolOSDiskSizeGB,
				VnetSubnetID: &mediumNodePoolVNetSubnetID,
				MaxPods:      to.Int32Ptr(30),
				Type:         containerservice.VirtualMachineScaleSets,
				Mode:         containerservice.System,
				StorageProfile: &containerservice.ManagedClusterAgentPoolProfileStorageProfile{
					OsDisk: &containerservice.ManagedClusterAgentPoolProfileStorageProfileOsDisk{
						DiskSizeGB: &mediumNodePoolOSDiskSizeGB,
						OsType:     containerservice.Linux,
						ManagedDisk: &containerservice.ManagedClusterAgentPoolProfileStorageProfileManagedDisk{
							StorageAccountType: containerservice.StorageAccountTypePremiumLRS,
						},
					},
				},
			},
			{
				Name:         &smallNodePoolName,
				Count:        &smallNodePoolCount,
				VMSize:       &smallNodePoolVMSize,
				OsDiskSizeGB: &smallNodePoolOSDiskSizeGB,
				VnetSubnetID: &smallNodePoolVNetSubnetID,
				MaxPods:      to.Int32Ptr(30),
				Type:         containerservice.VirtualMachineScaleSets,
				Mode:         containerservice.System,
				StorageProfile: &containerservice.ManagedClusterAgentPoolProfileStorageProfile{
					OsDisk: &containerservice.ManagedClusterAgentPoolProfileStorageProfileOsDisk{
						DiskSizeGB: &smallNodePoolOSDiskSizeGB,
						OsType:     containerservice.Linux,
						ManagedDisk: &containerservice.ManagedClusterAgentPoolProfileStorageProfileManagedDisk{
							StorageAccountType: containerservice.StorageAccountTypePremiumLRS,
						},
					},
				},
			},
		},
	}

	// Create cluster
	cluster, err := aksClient.CreateOrUpdate(ctx, resourceGroupName, clusterName, containerservice.ManagedCluster{
		Location:                 &location,
		ManagedClusterProperties: clusterProperties,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create AKS cluster: %v", err)
	}

	return &cluster, nil
}

func createKubernetesProvider(ctx context.Context, plumiContext *schema.ResourceData, cluster *containerservice.ManagedCluster) (*kubernetes.Provider, error) {
	// Get AKS credentials
	aksCredentials, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to get AKS credentials: %v", err)
	}

	// Define provider properties
	providerProperties := map[string]interface{}{
		"host": fmt.Sprintf("https://%s", *cluster.Fqdn),
		"client_certificate": []map[string]string{
			{
				"client_certificate":     *cluster.ManagedClusterProperties.ManagedClusterAPIServerAccessProfile.KubeConfig[0].ClientCertificateData,
				"client_key":             *cluster.ManagedClusterProperties.ManagedClusterAPIServerAccessProfile.KubeConfig[0].ClientKeyData,
				"cluster_ca_certificate": *cluster.ManagedClusterProperties.ManagedClusterAPIServerAccessProfile.KubeConfig[0].ClusterCACertificateData,
			},
		},
		"load_config_file": false,
	}

	// Create provider
	provider, err := kubernetes.NewProvider(kubernetes.ProviderConfig{
		Host: *cluster.Fqdn,
		ClientConfig: &clientcmdapi.Config{
			Clusters: map[string]*clientcmdapi.Cluster{
				*cluster.Name: {
					Server:                   fmt.Sprintf("https://%s", *cluster.Fqdn),
					CertificateAuthorityData: []byte(*cluster.ManagedClusterProperties.ManagedClusterAPIServerAccessProfile.KubeConfig[0].ClusterCACertificateData),
				},
			},
			AuthInfos: map[string]*clientcmdapi.AuthInfo{
				*cluster.Name: {
					ClientCertificateData: []byte(*cluster.ManagedClusterProperties.ManagedClusterAPIServerAccessProfile.KubeConfig[0].ClientCertificateData),
					ClientKeyData:         []byte(*cluster.ManagedClusterProperties.ManagedClusterAPIServerAccessProfile.KubeConfig[0].ClientKeyData),
				},
			},
			Contexts: map[string]*clientcmdapi.Context{
				*cluster.Name: {
					Cluster:  *cluster.Name,
					AuthInfo: *cluster.Name,
				},
			},
			CurrentContext: *cluster.Name,
		},
		LoadConfigFile: false,
	}, providerProperties)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes provider: %v", err)
	}

	return provider, nil
}

*/
