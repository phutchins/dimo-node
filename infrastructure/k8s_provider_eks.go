package infrastructure

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/eks"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Example can be found here: https://github.com/scottslowe/learning-tools/blob/main/pulumi/eks-from-scratch/main.go

func CreateEKSKubernetesCluster(ctx *pulumi.Context, projectName string, location string) (*eks.Cluster, error) {
	createIam(ctx)

	// Create a Security Group that we can use to actually connect to our cluster
	clusterSg, err := ec2.NewSecurityGroup(ctx, "cluster-sg", &ec2.SecurityGroupArgs{
		VpcId: vpcId,
		Egress: ec2.SecurityGroupEgressArray{
			ec2.SecurityGroupEgressArgs{
				Protocol:   pulumi.String("-1"),
				FromPort:   pulumi.Int(0),
				ToPort:     pulumi.Int(0),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Ingress: ec2.SecurityGroupIngressArray{
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(80),
				ToPort:     pulumi.Int(80),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
			ec2.SecurityGroupIngressArgs{
				Protocol:   pulumi.String("tcp"),
				FromPort:   pulumi.Int(443),
				ToPort:     pulumi.Int(443),
				CidrBlocks: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	// Create the EKS cluster
	cluster, err := eks.NewCluster(ctx, projectName, &eks.ClusterArgs{
		Name:    pulumi.String(projectName),
		RoleArn: eksClusterRoleArn,
		VpcConfig: &eks.ClusterVpcConfigArgs{
			EndpointPrivateAccess: pulumi.Bool(false),
			EndpointPublicAccess:  pulumi.Bool(true),
			SecurityGroupIds:      pulumi.StringArray{clusterSg.ID()},
			SubnetIds:             privateSubnets,
		},
		Tags: pulumi.StringMap{
			"Name": pulumi.String(projectName),
		},
		/*
			    ScalingConfig: &eks.NodeGroupScalingConfigArgs{
						DesiredSize: pulumi.Int(3),
						MinSize:     pulumi.Int(1),
						MaxSize:     pulumi.Int(5),
					}, */
	})
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

func CreateEKSKubernetesNodePools(ctx *pulumi.Context, projectName string, cluster *eks.Cluster, location string) (err error) {
	/*
		managedPolicyArns := []string{
			"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
			"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
			"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
		}

			tmpJSON0, err := json.Marshal(map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					map[string]interface{}{
						"Action": "sts:AssumeRole",
						"Effect": "Allow",
						"Sid":    nil,
						"Principal": map[string]interface{}{
							"Service": "ec2.amazonaws.com",
						},
					},
				},
			})
			if err != nil {
				return err
			}
			json0 := string(tmpJSON0)
			assumeRolePolicy := json0
			role1, err := iam.NewRole(ctx, "role1", &iam.RoleArgs{
				AssumeRolePolicy:  pulumi.String(assumeRolePolicy),
				ManagedPolicyArns: pulumi.ToStringArray(managedPolicyArns),
			})
			if err != nil {
				return err
			}
			role2, err := iam.NewRole(ctx, "role2", &iam.RoleArgs{
				AssumeRolePolicy:  pulumi.String(assumeRolePolicy),
				ManagedPolicyArns: pulumi.ToStringArray(managedPolicyArns),
			})
			if err != nil {
				return err
			}
			_, err = iam.NewInstanceProfile(ctx, "instanceProfile1", &iam.InstanceProfileArgs{
				Role: role1.Name,
			})
			if err != nil {
				return err
			}
			_, err = iam.NewInstanceProfile(ctx, "instanceProfile2", &iam.InstanceProfileArgs{
				Role: role2.Name,
			})
			if err != nil {
				return err
			}
	*/

	// Create the medium node pool
	// Create an EKS node pool

	_, err = eks.NewNodeGroup(ctx, projectName+"-small", &eks.NodeGroupArgs{
		ClusterName:   cluster.Name,
		InstanceTypes: pulumi.StringArray{pulumi.String("t2.small")},
		ScalingConfig: &eks.NodeGroupScalingConfigArgs{
			DesiredSize: pulumi.Int(3),
			MinSize:     pulumi.Int(1),
			MaxSize:     pulumi.Int(5),
		},
		//InstanceProfile: instanceProfile1.Name,
		/*
			Labels: map[string]string{
				"ondemand": "true",
			},
		*/
	})
	if err != nil {
		return err
	}

	_, err = eks.NewNodeGroup(ctx, projectName+"-medium", &eks.NodeGroupArgs{
		ClusterName:   cluster.Name,
		InstanceTypes: pulumi.StringArray{pulumi.String("t2.medium")},
		ScalingConfig: &eks.NodeGroupScalingConfigArgs{
			DesiredSize: pulumi.Int(3),
			MinSize:     pulumi.Int(1),
			MaxSize:     pulumi.Int(5),
		},
		/*
			Labels: map[string]string{
				"preemptible": "true",
			},
		*/
	})
	if err != nil {
		return err
	}

	return nil
}

func NewEKSKubernetesProvider(ctx *pulumi.Context, cluster *eks.Cluster) (*kubernetes.Provider, error) {
	// Create a kubeconfig string
	//masterAuth := cluster.MasterAuth.ClusterCaCertificate()
	kubeConfig := pulumi.All(cluster.Name, cluster.Endpoint, cluster.CertificateAuthority).ApplyT(func(args []interface{}) (pulumi.StringOutput, error) {
		certificateAuthority := args[2].(eks.ClusterCertificateAuthorityOutput)
		clusterCaCertificate := certificateAuthority.Data().Elem()
		clusterEndpoint := args[1].(string)
		clusterName := args[0].(string)

		kubeConfig := generateEKSKubeconfig(
			clusterEndpoint,
			clusterCaCertificate,
			clusterName,
		)

		return kubeConfig, nil
	})

	kubeProvider, err := kubernetes.NewProvider(ctx, "EKSk8sProvider", &kubernetes.ProviderArgs{
		Kubeconfig: kubeConfig.(pulumi.StringOutput),
	})
	if err != nil {
		return nil, err
	}

	// Return kubernetesProvider
	return kubeProvider, nil
}

// Create the KubeConfig structure as per https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html
func generateEKSKubeconfig(clusterEndpoint string, certData pulumi.StringOutput, clusterName string) pulumi.StringOutput {
	return pulumi.Sprintf(`{
"apiVersion": "v1",
"clusters": [{
		"cluster": {
				"server": "%s",
				"certificate-authority-data": "%s"
		},
		"name": "kubernetes",
}],
"contexts": [{
		"context": {
				"cluster": "kubernetes",
				"user": "aws",
		},
		"name": "aws",
}],
"current-context": "aws",
"kind": "Config",
"users": [{
		"name": "aws",
		"user": {
				"exec": {
						"apiVersion": "client.authentication.k8s.io/v1beta1",
						"command": "aws-iam-authenticator",
						"args": [
								"token",
								"-i",
								"%s",
						],
				},
		},
}],
    }`, clusterEndpoint, certData, clusterName)
}
