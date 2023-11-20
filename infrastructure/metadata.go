package infrastructure

import (
	gcpCompute "github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func AddSSHKeysMetadata(ctx *pulumi.Context, cloudProvider string, pubKey string, privKey string) error {
	switch cloudProvider {
	case "gcp":
		return addSSHKeysMetadataGCP(ctx, pubKey, privKey)
	case "aws":
		return addSSHKeysMetadataAWS(ctx, pubKey, privKey)
	default:
		return nil
	}

}

func addSSHKeysMetadataGCP(ctx *pulumi.Context, pubKey string, privKey string) (err error) {
	// Create a GCP network
	_, err = gcpCompute.NewProjectMetadata(ctx, "ssh-keys", &gcpCompute.ProjectMetadataArgs{
		Metadata: pulumi.StringMap{
			"ssh-keys": pulumi.Sprintf("%s:%s", sshUser, privKey),
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func addSSHKeysMetadataAWS(ctx *pulumi.Context, pubKey string, privKey string) (err error) {
	// Might not need to do this for AWS?
	return nil
}
