package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dimo/dimo-node/utils"
)

const helpText = `Password Manager for DIMO Infrastructure

Usage:
  password-manager [command] [flags]

Available Commands:
  add         Add or update a password configuration
  update      Update passwords in both Pulumi config and GCP Secret Manager
  get         Get a password for a specific service
  list        List all password configurations
  delete      Delete a password configuration
  compare     Compare passwords between GCP Secret Manager and External Secrets
  help        Show this help message

Flags for add:
  --stack string         Stack name (required)
  --service string       Service name (required)
  --length int          Password length (required)
  --special bool        Use special characters (default true)
  --gcp-secret string   GCP Secret ID (required)
  --k8s-secret string   Kubernetes secret name (required)
  --k8s-namespace string Kubernetes namespace (required)

Flags for update:
  --stack string        Stack name (required)
  --service string      Service name (optional, updates all if not specified)

Flags for get:
  --stack string        Stack name (required)
  --service string      Service name (required)

Flags for list:
  --stack string        Stack name (required)

Flags for delete:
  --stack string        Stack name (required)
  --service string      Service name (required)

Flags for compare:
  --stack string        Stack name (required)
  --service string      Service name (required)

Examples:
  # Add a new password configuration
  password-manager add --stack dimo-eu --service postgres-root --length 32 --gcp-secret postgres-root-password --k8s-secret postgres-root-secret --k8s-namespace default

  # Update all passwords
  password-manager update --stack dimo-eu

  # Update single password
  password-manager update --stack dimo-eu --service postgres-root

  # Get a specific password
  password-manager get --stack dimo-eu --service postgres-root

  # List all configurations
  password-manager list --stack dimo-eu

  # Delete a configuration
  password-manager delete --stack dimo-eu --service postgres-root

  # Compare passwords
  password-manager compare --stack dimo-eu --service postgres-root
`

func showHelp() {
	fmt.Println(helpText)
}

func main() {
	// Define command-line flags for each command
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	addStack := addCmd.String("stack", "", "Stack name (required)")
	addService := addCmd.String("service", "", "Service name (required)")
	addLength := addCmd.Int("length", 32, "Password length")
	addSpecial := addCmd.Bool("special", true, "Use special characters")
	addGCPSecret := addCmd.String("gcp-secret", "", "GCP Secret ID (required)")
	addK8sSecret := addCmd.String("k8s-secret", "", "Kubernetes secret name (required)")
	addK8sNamespace := addCmd.String("k8s-namespace", "", "Kubernetes namespace (required)")

	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)
	updateStack := updateCmd.String("stack", "", "Stack name (required)")
	updateService := updateCmd.String("service", "", "Service name (optional)")

	getCmd := flag.NewFlagSet("get", flag.ExitOnError)
	getStack := getCmd.String("stack", "", "Stack name (required)")
	getService := getCmd.String("service", "", "Service name (required)")

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listStack := listCmd.String("stack", "", "Stack name (required)")

	deleteCmd := flag.NewFlagSet("delete", flag.ExitOnError)
	deleteStack := deleteCmd.String("stack", "", "Stack name (required)")
	deleteService := deleteCmd.String("service", "", "Service name (required)")

	compareCmd := flag.NewFlagSet("compare", flag.ExitOnError)
	compareStack := compareCmd.String("stack", "", "Stack name (required)")
	compareService := compareCmd.String("service", "", "Service name (required)")

	// Check if a subcommand is provided
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(1)
	}

	// Parse subcommands
	switch os.Args[1] {
	case "help":
		showHelp()

	case "add":
		addCmd.Parse(os.Args[2:])
		if *addStack == "" || *addService == "" || *addGCPSecret == "" || *addK8sSecret == "" || *addK8sNamespace == "" {
			log.Fatal("Stack name, service name, GCP secret ID, K8s secret name, and K8s namespace are required")
		}
		config := utils.PasswordConfig{
			ServiceName:   *addService,
			Length:        *addLength,
			UseSpecial:    *addSpecial,
			GCPSecretID:   *addGCPSecret,
			K8sSecretName: *addK8sSecret,
			K8sNamespace:  *addK8sNamespace,
		}
		if err := utils.AddPasswordConfig(*addStack, config); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Added/updated password configuration for %s\n", *addService)

	case "update":
		updateCmd.Parse(os.Args[2:])
		if *updateStack == "" {
			log.Fatal("Stack name is required")
		}
		if err := utils.UpdatePasswords(*updateStack, *updateService); err != nil {
			log.Fatal(err)
		}
		fmt.Println("Passwords updated successfully")

	case "get":
		getCmd.Parse(os.Args[2:])
		if *getStack == "" || *getService == "" {
			log.Fatal("Stack name and service name are required")
		}
		password, err := utils.GetPassword(*getStack, *getService)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(password)

	case "list":
		listCmd.Parse(os.Args[2:])
		if *listStack == "" {
			log.Fatal("Stack name is required")
		}
		configs, err := utils.ListPasswordConfigs(*listStack)
		if err != nil {
			log.Fatal(err)
		}
		configJSON, err := json.MarshalIndent(configs, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(configJSON))

	case "delete":
		deleteCmd.Parse(os.Args[2:])
		if *deleteStack == "" || *deleteService == "" {
			log.Fatal("Stack name and service name are required")
		}
		if err := utils.DeletePasswordConfig(*deleteStack, *deleteService); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Deleted password configuration for %s\n", *deleteService)

	case "compare":
		compareCmd.Parse(os.Args[2:])
		if *compareStack == "" || *compareService == "" {
			log.Fatal("Stack name and service name are required")
		}
		comparison, err := utils.ComparePasswords(*compareStack, *compareService)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("\nPassword Comparison for %s:\n", comparison.ServiceName)
		fmt.Printf("Match: %v\n", comparison.Match)
		fmt.Printf("GCP Secret Length: %d\n", comparison.GCPLength)
		fmt.Printf("K8s Secret Length: %d\n", comparison.K8sLength)
		if !comparison.Match {
			fmt.Println("\nWARNING: Passwords do not match!")
		}

	default:
		fmt.Printf("Unknown command: %s\n\n", os.Args[1])
		showHelp()
		os.Exit(1)
	}
}
