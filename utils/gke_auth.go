package utils

import (
	"os"
	"os/exec"
)

func EnsureGKEAuth() error {
	// Set required environment variable
	os.Setenv("USE_GKE_GCLOUD_AUTH_PLUGIN", "True")

	// Check if plugin is installed
	_, err := exec.LookPath("gke-gcloud-auth-plugin")
	if err != nil {
		// Install the plugin using gcloud
		cmd := exec.Command("gcloud", "components", "install", "gke-gcloud-auth-plugin")
		err = cmd.Run()
		if err != nil {
			return err
		}

		// Update kubeconfig
		cmd = exec.Command("gcloud", "container", "clusters", "get-credentials",
			os.Getenv("CLUSTER_NAME"),
			"--region", os.Getenv("CLUSTER_REGION"),
			"--project", os.Getenv("PROJECT_ID"))
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	return nil
}
