package infrastructure

import (
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/container"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateEKSCluster(ctx *pulumi.Context, projectName string, location string) (*container.Cluster, error) {
	return nil, nil
}
