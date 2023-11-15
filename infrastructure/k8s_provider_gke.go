package infrastructure

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

	return cluster, nil
}

func NewKubernetesProvider(ctx *pulumi.Context, cluster *container.Cluster) (*kubernetes.Provider, error) {
	// Create a kubeconfig string
	kubeconfig := pulumi.All(cluster.Name, cluster.Endpoint, cluster.MasterAuth).ApplyT(func(args []interface{}) (string, error) {
		clusterName := args[0].(string)
		endpoint := args[1].(string)
		masterAuth := args[2].(*container.ClusterMasterAuth)
		clusterCaCertificate := *masterAuth.ClusterCaCertificate

		return fmt.Sprintf(`
apiVersion: v1
clusters:
- cluster:
	certificate-authority-data: %s
	server: https://%s
name: %s
contexts:
- context:
	cluster: %s
	user: %s
name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: %s
user:
	auth-provider:
		config:
			cmd-args: config config-helper --format=json
			cmd-path: gcloud
			expiry-key: '{.credential.token_expiry}'
			token-key: '{.credential.access_token}'
		name: gcp
`, clusterCaCertificate, endpoint, clusterName, clusterName, clusterName, clusterName, clusterName, clusterName), nil
	}).(pulumi.StringOutput)

	// Create the Kubernetes provider
	kubeProvider, err := kubernetes.NewProvider(ctx, "k8sProvider", &kubernetes.ProviderArgs{
		Kubeconfig: kubeconfig,
	})
	if err != nil {
		return nil, err
	}

	return kubeProvider, nil
}
