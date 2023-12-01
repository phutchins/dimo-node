# DIMO Node
Deployment and management of DIMO nodes and related infrastructure

This repository brings together all of the components required to deploy and manage a DIMO node. Deployment and management scripts are brought into this repository as submodules. You can see the submodules below in the directory structure.

## Directory Structure
```
├── infrastructure
│   ├── cloud
│   │   ├── aws
│   │   ├── azure
│   │   └── gcp
│   └── local
├── dependencies
│   ├── postgres-operator
│   ├── postgres-operator-ui
│   ├── monitoring (stuff)
├── applications
│   ├── dimo-identity
├── scripts
├── keys
├── README.md
├── Makefile
└── .gitignore

```

## Prerequisites
Prequisites will be handled by the deployment management scripts automatically for both local and cloud deployments but for now, please ensure you have the following installed:
- [Pulumi](https://www.pulumi.com/docs/install/)

### Educate Yourself
To better understand the DIMO node and its components, please review the following documentation:
- [DIMO Node Cluster Architecture](https://asdf.com/asdf.html)
- [DIMO Core Concepts](https://asdf.com/asdf.html)

## Getting Started
You have a few options for getting started with this repository. You can either clone the repository and run the deployment management scripts locally, or you can use the deployment management scripts to deploy a DIMO node to a cloud provider.

## Cloud Dev Deployment
To get started, clone this repository and run the following commands:

Generate ssh keys for the DIMO node (key is used for ssh access to the K3s VM)
```
ssh-keygen -t rsa -b 4096 -C "pulumi" -f keys/pulumi_key -N ""
```

Configure which cloud provider you want to deploy to and the type of deployment
```
pulumi config set cloud-provider <cloud-provider> (ex: gcp | aws)
pulumi config set deployment-type <deployment-type> (ex: gke | eks | k3s)
```

Acceptable Option Combinations
- gcp / gke
- aws / eks
- gcp / k3s

NOTE: If k3s deployment has only been tested with Google Cloud

### Google Cloud
Ensure you have the following IAM roles for your GCP user (or service account)
- Compute Admin
- Kubernetes Engine Admin
- Service Account Admin
- Service Account User
- Storage Admin

Authenticate to Google Cloud
```
gcloud auth login
gcloud config set project <project-name>
gcloud auth application-default login
```

Initialize Pulumi
```
# This will ask you to name the stack, you can use the default <dev> name
pulumi stack init
```

Set the following Pulumi configuration variables:
(defaults set to Google Cloud)
```
pulumi config set gcp:project <project-name> (ex: dimo-dev-401815)
pulumi config set gcp:zone <zone> (ex: us-central1-a)
pulumi config set gcp:region <region> (ex: us-central1)
```

Deploy the DIMO node
```
# Use -y to automatically respond yes
pulumi up
```

### Amazon Web Services
Ensure you have the following IAM roles for your AWS user (or service account)
- AmazonEC2FullAccess
- AmazonVPCFullAccess
- AmazonEKSClusterPolicy
- AmazonEKSWorkerNodePolicy
- AmazonEKS_CNI_Policy
- AmazonEKSServicePolicy
- AmazonEKSContainerRegistryPowerUser

Install the AWS IAM Authenticator
Doc: https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html
```
brew install aws-iam-authenticator
```

Authenticate to AWS
```
aws configure
```



### Cloud Production Deployment (coming soon)
To get started, clone this repository and run the following commands:
```
[coming soon]
```

## Deployment Management Scripts
```
[coming soon]
```

## Usefull Commands
Refresh the stacks state from the current resources
```
pulumi refresh
```

Get a decoded password from a secret
```
kubectl get secrets/db-user-pass --template={{.data.password}} | base64 -D
```

Create / Set a variable for a stack
```
pulumi stack select [stack-name] # if not already selected
pulumi config set <key> <value>
```

## Cluster Command Line Access
### Google Cloud
```
gcloud container clusters get-credentials <cluster-name> --zone <zone> --project <project-name>
```

### Amazon Web Services
#### Prerequisites
Install the AWS IAM Authenticator
Doc: https://docs.aws.amazon.com/eks/latest/userguide/install-aws-iam-authenticator.html
```
brew install aws-iam-authenticator
```

Authenticate to AWS
```
aws configure
```

### Get Cluster Credentials
```
aws eks --region <region> update-kubeconfig --name <cluster-name>
```

## Managing Multiple Stacks
[coming soon]

# Troubleshooting & Debugging
## Pulumi
Running Pulumi in debug mode
```
pulumi up --logtostderr --logflow -v=10 2> out.txt
```

List resources in stack with IDs
```
pulumi stack -i
```



# Helpful Links
## Infrastructure
### Pulumi General
- [Pulumi](https://www.pulumi.com/docs/)
- [Pulumi GCP Provider](https://www.pulumi.com/docs/reference/pkg/gcp/)
- [Pulumi Organizational Paterns](https://www.pulumi.com/blog/organizational-patterns-infra-repo/)
### Pulumi Examples
- [EKS From Scratch](https://github.com/scottslowe/learning-tools/blob/main/pulumi/eks-from-scratch/vpc.go)
- [Pulumi Self Hosted Installers](https://github.com/pulumi/pulumi-self-hosted-installers/blob/master/ecs-hosted/go/infrastructure/main.go)
### Pulumi Golang
- [Using Go Generics in Pulumi](https://www.pulumi.com/blog/go-generics-preview/)
## Dependencies
### Database / Postgres
- [Postgres Operator](https://postgres-operator.readthedocs.io/en/latest/)
- [Postgres Cluster Creation/Configuration](https://github.com/zalando/postgres-operator/blob/master/manifests/complete-postgres-manifest.yaml)