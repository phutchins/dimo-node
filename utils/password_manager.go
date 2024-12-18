package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
)

type PasswordConfig struct {
	ServiceName string
	Length      int
	UseSpecial  bool
}

var defaultConfigs = []PasswordConfig{
	{ServiceName: "postgres-root", Length: 32, UseSpecial: true},
	{ServiceName: "identity-api-db", Length: 32, UseSpecial: true},
	{ServiceName: "prometheus-grafana", Length: 16, UseSpecial: true},
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

func createLocalWorkspace(ctx context.Context) (auto.Workspace, error) {
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

// UpdatePasswords updates passwords in Pulumi config and Secret Manager
func UpdatePasswords(stack string) error {
	ctx := context.Background()

	// Create workspace
	ws, err := createLocalWorkspace(ctx)
	if err != nil {
		return err
	}

	// Select the stack
	s, err := auto.SelectStack(ctx, stack, ws)
	if err != nil {
		return fmt.Errorf("failed to select stack: %v", err)
	}

	// Generate and update passwords for each service
	for _, config := range defaultConfigs {
		password, err := GenerateSecurePassword(config.Length, config.UseSpecial)
		if err != nil {
			return fmt.Errorf("failed to generate password for %s: %v", config.ServiceName, err)
		}

		// Update the config with the new password
		configKey := fmt.Sprintf("passwords.%s", config.ServiceName)
		err = s.SetConfig(ctx, configKey, auto.ConfigValue{
			Value:  password,
			Secret: true,
		})
		if err != nil {
			return fmt.Errorf("failed to set config for %s: %v", config.ServiceName, err)
		}
	}

	return nil
}

// GetPassword retrieves a password from Pulumi config
func GetPassword(stack, serviceName string) (string, error) {
	ctx := context.Background()

	// Create workspace
	ws, err := createLocalWorkspace(ctx)
	if err != nil {
		return "", err
	}

	// Select the stack
	s, err := auto.SelectStack(ctx, stack, ws)
	if err != nil {
		return "", fmt.Errorf("failed to select stack: %v", err)
	}

	// Get the config value
	configKey := fmt.Sprintf("passwords.%s", serviceName)
	val, err := s.GetConfig(ctx, configKey)
	if err != nil {
		return "", fmt.Errorf("failed to get config: %v", err)
	}

	if val.Value == "" {
		return "", fmt.Errorf("password not found for service: %s", serviceName)
	}

	return val.Value, nil
}
