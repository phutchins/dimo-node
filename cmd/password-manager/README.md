# Password Manager for DIMO Infrastructure

This tool manages passwords for various services in the DIMO infrastructure using Pulumi's configuration system and Google Cloud Secret Manager.

## Features

- Generates cryptographically secure passwords
- Stores passwords in Pulumi configuration as secrets
- Creates External Secrets in Kubernetes to access passwords
- Supports multiple services (PostgreSQL root, Identity API DB, etc.)

## Usage

### Building the tool

```bash
go build -o password-manager cmd/password-manager/main.go
```

### Updating passwords

To generate new passwords for all services in a stack:

```bash
./password-manager update --stack dimo-eu
```

### Getting a specific password

To retrieve a password for a specific service:

```bash
./password-manager get --stack dimo-eu --service postgres-root
```

## Available Services

- `postgres-root`: PostgreSQL root user password
- `identity-api-db`: Identity API database user password
- `prometheus-grafana`: Grafana UI password

## Integration with Pulumi

The passwords are stored in the Pulumi configuration and are automatically used to create External Secrets in Kubernetes during deployment. The External Secrets operator then creates Kubernetes secrets that can be used by applications.

## Adding New Services

To add a new service that requires password management:

1. Add a new entry to `defaultConfigs` in `utils/password_manager.go`
2. Update the `createDatabaseSecrets` function in `dependencies/secrets.go` to create the corresponding External Secret
3. Update your application's Pulumi code to use the new secret

## Security Considerations

- Passwords are stored as secrets in Pulumi configuration
- Passwords are never logged or displayed in plain text
- External Secrets are used to securely distribute passwords to applications
- Passwords can be rotated by running the update command 