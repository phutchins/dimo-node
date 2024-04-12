package infrastructure

import (
	"fmt"

	//"github.com/pulumi/pulumi-gcp/sdk/v5/go/gcp/container"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/container"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Consider
// ignoreChanges: ["verticalPodAutoscaling"],
var nodeLocations = []string{
	"us-central1-a",
	//"us-central1-b",
	//"us-central1-c",
}

var oauthScopes = []string{
	"https://www.googleapis.com/auth/monitoring",
	"https://www.googleapis.com/auth/logging.write",
}

func CreateGKECluster(ctx *pulumi.Context, projectName string, location string) (*container.Cluster, error) {
	// Create the GKE cluster
	// Array of node locations

	cluster, err := container.NewCluster(ctx, projectName, &container.ClusterArgs{
		InitialNodeCount: pulumi.Int(2),
		//RemoveDefaultNodePool: pulumi.Bool(true),
		Location: pulumi.String("us-central1-a"),
		//Location:         pulumi.String(location),
		MinMasterVersion: pulumi.String("latest"),
		Network:          Network.ID(),
		Subnetwork:       Subnetwork.ID(),
		//NodeLocations:    pulumi.ToStringArray(nodeLocations),
		NodeConfig: &container.ClusterNodeConfigArgs{
			MachineType: pulumi.String("n1-standard-1"), // TODO: Make this dynamic
			DiskSizeGb:  pulumi.Int(30),                 // TODO: Make this dynamic/configurable
			OauthScopes: pulumi.ToStringArray(oauthScopes),
			Preemptible: pulumi.Bool(false),
		},
	})
	if err != nil {
		return nil, err
	}

	//ctx.Export("kubeconfig", generateKubeconfig(cluster.Endpoint, cluster.Name, cluster.MasterAuth))
	ctx.Export("cluster.MasterAuth", cluster.MasterAuth)
	ctx.Export("clusterEndpoint", cluster.Endpoint)

	return cluster, nil
}

func CreateGKENodePools(ctx *pulumi.Context, projectName string, cluster *container.Cluster, location string) (err error) {
	// Create the medium node pool
	/*
		_, err = container.NewNodePool(ctx, projectName+"-medium", &container.NodePoolArgs{
			Cluster:       cluster.Name,
			Location:      pulumi.String(location),
			NodeCount:     pulumi.Int(1),
			NodeLocations: pulumi.ToStringArray(nodeLocations),
			NodeConfig: &container.NodePoolNodeConfigArgs{
				MachineType: pulumi.String("n1-standard-2"),
				DiskSizeGb:  pulumi.Int(30),
			},
		})
		if err != nil {
			return err
		}
	*/

	// Create the small node pool
	_, err = container.NewNodePool(ctx, projectName+"-small", &container.NodePoolArgs{
		Cluster:  cluster.Name,
		Location: pulumi.String("us-central1-a"),
		//Location:      pulumi.String(location),
		NodeCount:     pulumi.Int(1),
		NodeLocations: pulumi.ToStringArray(nodeLocations),
		NodeConfig: &container.NodePoolNodeConfigArgs{
			MachineType: pulumi.String("n1-standard-1"),
			DiskSizeGb:  pulumi.Int(30),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func NewGKEKubernetesProvider(ctx *pulumi.Context, cluster *container.Cluster) (*kubernetes.Provider, error) {
	// Create a kubeconfig string
	//masterAuth := cluster.MasterAuth.ClusterCaCertificate()
	kubeConfig := pulumi.All(cluster.Name, cluster.Endpoint, cluster.MasterAuth).ApplyT(func(args []interface{}) (pulumi.StringOutput, error) {
		masterAuth := args[2].(container.ClusterMasterAuth)
		//fmt.Printf("(pall) masterAuth: %v\n", masterAuth)

		//bytes := []byte(*masterAuth.ClusterCaCertificate)

		// encode the byte slice in base64
		//clusterCaCertificate := base64.StdEncoding.EncodeToString(bytes)
		//clusterCaCertificate := masterAuth.ClusterCaCertificate
		clusterCaCertificate := *masterAuth.ClusterCaCertificate
		//fmt.Printf("(pall) clusterCaCertificate: %s\n", clusterCaCertificate)

		clusterEndpoint := args[1].(string)
		//fmt.Printf("(pall) clusterEndpoint: %s\n", clusterEndpoint)

		clusterName := args[0].(string)
		//fmt.Printf("(pall) clusterName: %s\n", clusterName)

		kubeConfig := generateKubeconfig(
			clusterEndpoint,
			clusterName,
			clusterCaCertificate,
		)

		//fmt.Printf("(pall) Args[0]: %v\n", args[0])
		//fmt.Printf("(pall) Args[1]: %v\n", args[1])

		kubeConfig.ApplyT(func(s string) string {
			//fmt.Printf("(pall) kubeConfig: %s\n", s)
			return s
		})

		//fmt.Printf("(pall) KubeConfig: %v\n", kubeConfig)
		//fmt.Printf("Args[2]: %v", args[2])
		//masterAuth := args[2].(container.ClusterMasterAuth)
		//clusterCaCertificate := masterAuth.ClusterCaCertificate
		//fmt.Printf("Args[2].ClusterCaCertificate: %v", clusterCaCertificate)

		return kubeConfig, nil
	})

	kubeProvider, err := kubernetes.NewProvider(ctx, "GKEk8sProvider", &kubernetes.ProviderArgs{
		Kubeconfig: kubeConfig.(pulumi.StringOutput),
	})
	if err != nil {
		return nil, err
	}

	// Return kubernetesProvider
	return kubeProvider, nil
}

func generateKubeconfig(clusterEndpoint string, clusterName string,
	clusterCaCertificate string) pulumi.StringOutput {
	//context := pulumi.Sprintf("dimo_%s", clusterName).ToStringOutput()
	context := clusterName

	//fmt.Printf("(gen config) clusterCaCertificate: %s\n", clusterCaCertificate)
	//fmt.Printf("(gen config) clusterEndpoint: %s\n", clusterEndpoint)

	kubeConfig := fmt.Sprintf(`apiVersion: v1
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
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: gke-gcloud-auth-plugin
      installHint: Install gke-gcloud-auth-plugin for use with kubectl by following
        https://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke
      provideClusterInfo: true`,
		clusterCaCertificate, clusterEndpoint, context, context, context, context, context, context)

	//kubeConfig = strings.Replace(kubeConfig, "\t", "  ", -1)

	/*
		pulumi.String(kubeConfig).ApplyT(func(s string) string {
			fmt.Printf("(gen config) kubeConfig: %s\n", s)
			return s
		})
	*/

	//fmt.Printf("(gen config) kubeConfig: %s\n", kubeConfig)
	//fmt.Printf("(gen config) kubeConfig: %s\n\n", kubeConfig.(pulumi.StringOutput))

	// Convert kubeConfig to pulumi.StringOutput
	kubeConfigOutput := pulumi.String(kubeConfig)

	return kubeConfigOutput.ToStringOutput()

	/*
	     return pulumi.Sprintf(`apiVersion: v1
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
	       exec:
	         apiVersion: client.authentication.k8s.io/v1beta1
	         command: gke-gcloud-auth-plugin
	         installHint: Install gke-gcloud-auth-plugin for use with kubectl by following
	           https://cloud.google.com/blog/products/containers-kubernetes/kubectl-auth-changes-in-gke
	         provideClusterInfo: true
	   `, clusterCaCertificate, clusterEndpoint, context, context, context, context, context, context)
	*/

}

/*
    kubeconfig := pulumi.All(cluster.Name, cluster.Endpoint, cluster.MasterAuth).ApplyT(func(args []interface{}) (string, error) {
      clusterName := args[0].(string)
      endpoint := args[1].(string)
      masterAuth := args[2].(container.ClusterMasterAuth)
      clusterCaCertificate := masterAuth.ClusterCaCertificate

      return fmt.Sprintf(`
  apiVersion: v1
  clusters:
  - cluster:
    certificate-authority-data: %s
    server: 'https://%s'
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
  `, *clusterCaCertificate, endpoint, clusterName, clusterName, clusterName, clusterName, clusterName, clusterName), nil
    }).(pulumi.StringOutput)
*/

//ctx.Export("kubeconfig", kubeconfig)

// Create the Kubernetes provider
