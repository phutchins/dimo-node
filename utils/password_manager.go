package utils

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

type PasswordConfig struct {
	ServiceName   string `json:"serviceName"`
	Length        int    `json:"length"`
	UseSpecial    bool   `json:"useSpecial"`
	GCPSecretID   string `json:"gcpSecretId"`   // GCP Secret Manager secret ID
	K8sSecretName string `json:"k8sSecretName"` // Kubernetes secret name
	K8sNamespace  string `json:"k8sNamespace"`  // Kubernetes namespace
}

// getPasswordConfigs retrieves password configurations from Pulumi stack config
func getPasswordConfigs(ctx context.Context, stack auto.Stack) (map[string]PasswordConfig, error) {
	// Try to get password configs from stack config
	val, err := stack.GetConfig(ctx, "password-configs")
	if err != nil || val.Value == "" {
		return make(map[string]PasswordConfig), nil
	}

	var configs map[string]PasswordConfig
	if err := json.Unmarshal([]byte(val.Value), &configs); err != nil {
		return nil, fmt.Errorf("failed to parse password configs: %v", err)
	}

	return configs, nil
}

// savePasswordConfigs saves password configurations to Pulumi stack config
func savePasswordConfigs(ctx context.Context, stack auto.Stack, configs map[string]PasswordConfig) error {
	configJSON, err := json.Marshal(configs)
	if err != nil {
		return fmt.Errorf("failed to marshal password configs: %v", err)
	}

	err = stack.SetConfig(ctx, "password-configs", auto.ConfigValue{
		Value: string(configJSON),
	})
	if err != nil {
		return fmt.Errorf("failed to save password configs: %v", err)
	}

	return nil
}

// AddPasswordConfig adds or updates a password configuration
func AddPasswordConfig(stack string, config PasswordConfig) error {
	ctx := context.Background()

	ws, err := CreateLocalWorkspace(ctx)
	if err != nil {
		return err
	}

	s, err := auto.SelectStack(ctx, stack, ws)
	if err != nil {
		return fmt.Errorf("failed to select stack: %v", err)
	}

	configs, err := getPasswordConfigs(ctx, s)
	if err != nil {
		return err
	}

	configs[config.ServiceName] = config
	return savePasswordConfigs(ctx, s, configs)
}

// DeletePasswordConfig removes a password configuration
func DeletePasswordConfig(stack, serviceName string) error {
	ctx := context.Background()

	ws, err := CreateLocalWorkspace(ctx)
	if err != nil {
		return err
	}

	s, err := auto.SelectStack(ctx, stack, ws)
	if err != nil {
		return fmt.Errorf("failed to select stack: %v", err)
	}

	configs, err := getPasswordConfigs(ctx, s)
	if err != nil {
		return err
	}

	delete(configs, serviceName)
	return savePasswordConfigs(ctx, s, configs)
}

// ListPasswordConfigs returns all password configurations
func ListPasswordConfigs(stack string) (map[string]PasswordConfig, error) {
	ctx := context.Background()

	ws, err := CreateLocalWorkspace(ctx)
	if err != nil {
		return nil, err
	}

	s, err := auto.SelectStack(ctx, stack, ws)
	if err != nil {
		return nil, fmt.Errorf("failed to select stack: %v", err)
	}

	return getPasswordConfigs(ctx, s)
}

// GenerateSecurePassword generates a cryptographically secure password
func GenerateSecurePassword(length int, useSpecial bool) (string, error) {
	const (
		letterBytes  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		numberBytes  = "0123456789"
		specialBytes = "!@#$%^&*()_+-=[]{}|;:,.<>?"
	)

	var charset string
	if useSpecial {
		charset = letterBytes + numberBytes + specialBytes
	} else {
		charset = letterBytes + numberBytes
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	return string(bytes), nil
}

// CreateLocalWorkspace creates a new Pulumi workspace
func CreateLocalWorkspace(ctx context.Context) (auto.Workspace, error) {
	// Get the current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %v", err)
	}

	// Find the project root (where Pulumi.yaml is located)
	projectRoot := workDir
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "Pulumi.yaml")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			return nil, fmt.Errorf("could not find Pulumi.yaml in any parent directory")
		}
		projectRoot = parent
	}

	// Create workspace
	ws, err := auto.NewLocalWorkspace(ctx, auto.WorkDir(projectRoot))
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %v", err)
	}

	return ws, nil
}

// GetGCPProjectID gets the GCP project ID from Pulumi config
func GetGCPProjectID(ctx context.Context, stack auto.Stack) (string, error) {
	val, err := stack.GetConfig(ctx, "gcp-project")
	if err != nil {
		return "", fmt.Errorf("failed to get GCP project ID from config: %v", err)
	}
	return val.Value, nil
}

// createOrUpdateSecret creates or updates a secret in GCP Secret Manager
func createOrUpdateSecret(ctx context.Context, client *secretmanager.Client, projectID, secretID, value string) error {
	secretPath := fmt.Sprintf("projects/%s/secrets/%s", projectID, secretID)

	// Try to access the secret first
	_, err := client.GetSecret(ctx, &secretmanagerpb.GetSecretRequest{
		Name: secretPath,
	})

	if err != nil {
		// Secret doesn't exist, create it
		createSecretReq := &secretmanagerpb.CreateSecretRequest{
			Parent:   fmt.Sprintf("projects/%s", projectID),
			SecretId: secretID,
			Secret: &secretmanagerpb.Secret{
				Replication: &secretmanagerpb.Replication{
					Replication: &secretmanagerpb.Replication_Automatic_{
						Automatic: &secretmanagerpb.Replication_Automatic{},
					},
				},
			},
		}
		_, err = client.CreateSecret(ctx, createSecretReq)
		if err != nil {
			return fmt.Errorf("failed to create secret: %v", err)
		}
	}

	// Add new version with the secret value
	addSecretVersionReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretPath,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	}
	_, err = client.AddSecretVersion(ctx, addSecretVersionReq)
	if err != nil {
		return fmt.Errorf("failed to add secret version: %v", err)
	}

	return nil
}

// UpdatePasswords updates passwords in both Pulumi config and GCP Secret Manager
func UpdatePasswords(stack string, serviceName string) error {
	ctx := context.Background()

	// Create Secret Manager client
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create Secret Manager client: %v", err)
	}
	defer client.Close()

	// Create workspace
	ws, err := CreateLocalWorkspace(ctx)
	if err != nil {
		return err
	}

	// Select the stack
	s, err := auto.SelectStack(ctx, stack, ws)
	if err != nil {
		return fmt.Errorf("failed to select stack: %v", err)
	}

	// Get GCP project ID
	projectID, err := GetGCPProjectID(ctx, s)
	if err != nil {
		return err
	}

	// Get password configurations
	configs, err := getPasswordConfigs(ctx, s)
	if err != nil {
		return err
	}

	// If serviceName is provided, only update that service
	if serviceName != "" {
		config, exists := configs[serviceName]
		if !exists {
			return fmt.Errorf("no configuration found for service: %s", serviceName)
		}
		configs = map[string]PasswordConfig{serviceName: config}
	}

	// Generate and update passwords for each service
	for _, config := range configs {
		password, err := GenerateSecurePassword(config.Length, config.UseSpecial)
		if err != nil {
			return fmt.Errorf("failed to generate password for %s: %v", config.ServiceName, err)
		}

		// Update Pulumi config
		configKey := fmt.Sprintf("passwords.%s", config.ServiceName)
		err = s.SetConfig(ctx, configKey, auto.ConfigValue{
			Value:  password,
			Secret: true,
		})
		if err != nil {
			return fmt.Errorf("failed to set Pulumi config for %s: %v", config.ServiceName, err)
		}

		// Update GCP Secret Manager
		err = createOrUpdateSecret(ctx, client, projectID, config.GCPSecretID, password)
		if err != nil {
			return fmt.Errorf("failed to update GCP secret for %s: %v", config.ServiceName, err)
		}
	}

	return nil
}

// GetPassword retrieves a password from GCP Secret Manager
func GetPassword(stack, serviceName string) (string, error) {
	ctx := context.Background()

	// Get the service configuration first
	ws, err := CreateLocalWorkspace(ctx)
	if err != nil {
		return "", err
	}

	s, err := auto.SelectStack(ctx, stack, ws)
	if err != nil {
		return "", fmt.Errorf("failed to select stack: %v", err)
	}

	configs, err := getPasswordConfigs(ctx, s)
	if err != nil {
		return "", err
	}

	config, exists := configs[serviceName]
	if !exists {
		return "", fmt.Errorf("no configuration found for service: %s", serviceName)
	}

	// Create Secret Manager client
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create Secret Manager client: %v", err)
	}
	defer client.Close()

	// Get GCP project ID
	projectID, err := GetGCPProjectID(ctx, s)
	if err != nil {
		return "", err
	}

	// Access the latest version of the secret
	secretPath := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, config.GCPSecretID)

	accessRequest := &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretPath,
	}

	result, err := client.AccessSecretVersion(ctx, accessRequest)
	if err != nil {
		return "", fmt.Errorf("failed to access secret: %v", err)
	}

	return string(result.Payload.Data), nil
}

// GetPasswordFromExternalSecret retrieves a password from Kubernetes External Secret
func GetPasswordFromExternalSecret(stack, serviceName string) (string, error) {
	ctx := context.Background()

	// Get the service configuration first
	ws, err := CreateLocalWorkspace(ctx)
	if err != nil {
		return "", err
	}

	s, err := auto.SelectStack(ctx, stack, ws)
	if err != nil {
		return "", fmt.Errorf("failed to select stack: %v", err)
	}

	configs, err := getPasswordConfigs(ctx, s)
	if err != nil {
		return "", err
	}

	config, exists := configs[serviceName]
	if !exists {
		return "", fmt.Errorf("no configuration found for service: %s", serviceName)
	}

	// Load kubernetes configuration
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return "", fmt.Errorf("failed to load kubeconfig: %v", err)
	}

	// Create kubernetes clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	secret, err := clientset.CoreV1().Secrets(config.K8sNamespace).Get(context.Background(), config.K8sSecretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s in namespace %s: %v", config.K8sSecretName, config.K8sNamespace, err)
	}

	// Get password from secret data
	password, ok := secret.Data["password"]
	if !ok {
		return "", fmt.Errorf("password key not found in secret %s", config.K8sSecretName)
	}

	return string(password), nil
}

// ComparePasswords compares passwords between GCP Secret Manager and External Secret
func ComparePasswords(stack, serviceName string) (*PasswordComparison, error) {
	gcpPassword, err := GetPassword(stack, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get GCP password: %v", err)
	}

	k8sPassword, err := GetPasswordFromExternalSecret(stack, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get External Secret password: %v", err)
	}

	return &PasswordComparison{
		ServiceName: serviceName,
		GCPPassword: gcpPassword,
		K8sPassword: k8sPassword,
		Match:       gcpPassword == k8sPassword,
		GCPLength:   len(gcpPassword),
		K8sLength:   len(k8sPassword),
	}, nil
}

// PasswordComparison holds the comparison results between GCP and K8s passwords
type PasswordComparison struct {
	ServiceName string
	GCPPassword string
	K8sPassword string
	Match       bool
	GCPLength   int
	K8sLength   int
}
