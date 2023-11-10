package infrastructure

/*
import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/container"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateGKECluster(ctx *pulumi.Context, projectName string, location string) (*container.Cluster, error) {
	// Create the GKE cluster
	cluster, err := container.NewCluster(ctx, projectName, &container.ClusterArgs{
		InitialNodeCount: pulumi.Int(1),
		Location:         pulumi.String(location),
		MinMasterVersion: pulumi.String("latest"),
		NodeConfig: &container.ClusterNodeConfigArgs{
			MachineType: pulumi.String("n1-standard-4"),
			OauthScopes: pulumi.StringArray{
				pulumi.String("https://www.googleapis.com/auth/compute"),
			},
			Preemptible: pulumi.Bool(false),
		},
	})
	if err != nil {
		return nil, err
	}

	// Create the medium node pool
	_, err = container.NewNodePool(ctx, projectName+"-medium", &container.NodePoolArgs{
		Cluster:   cluster.Name,
		Location:  pulumi.String(location),
		NodeCount: pulumi.Int(1),
		NodeConfig: &container.NodePoolNodeConfigArgs{
			MachineType: pulumi.String("n1-standard-4"),
			DiskSizeGb:  pulumi.Int(100),
		},
	})
	if err != nil {
		return nil, err
	}

	// Create the small node pool
	_, err = container.NewNodePool(ctx, projectName+"-small", &container.NodePoolArgs{
		Cluster:   cluster.Name,
		Location:  pulumi.String(location),
		NodeCount: pulumi.Int(1),
		NodeConfig: &container.NodePoolNodeConfigArgs{
			MachineType: pulumi.String("n1-standard-2"),
			DiskSizeGb:  pulumi.Int(50),
		},
	})
	if err != nil {
		return nil, err
	}

	// Create the Kubernetes provider
	k8sProvider, err := NewKubernetesProvider(ctx, projectName, cluster.Endpoint, cluster.MasterAuth)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func NewKubernetesProvider(ctx *pulumi.Context, projectName string, endpoint pulumi.StringOutput, masterAuth container.ClusterMasterAuthOutput) (*kubernetes.Provider, error) {
	// Create the Kubernetes provider
	k8sProvider, err := kubernetes.NewProvider(ctx, projectName, &kubernetes.ProviderArgs{
		Host: endpoint.ApplyString(func(endpoint string) string {
			return fmt.Sprintf("https://%s", endpoint)
		}),
		Username: masterAuth.Username,
		Password: masterAuth.Password,
		ClientCertificate: &kubernetes.ProviderClientCertificateArgs{
			CertificateData: masterAuth.ClientCertificateConfig.Certificate,
			PrivateKeyData:  masterAuth.ClientCertificateConfig.PrivateKey,
		},
		ClusterCaCertificate: masterAuth.ClusterCaCertificate,
	})
	if err != nil {
		return nil, err
	}

	return k8sProvider, nil
}

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
*/
