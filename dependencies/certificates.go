package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallCertificates(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) error {
	// Install cert-manager and Let's Encrypt
	err := InstallLetsEncrypt(ctx, kubeProvider)
	if err != nil {
		return err
	}

	return nil
}
