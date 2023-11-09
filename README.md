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

### Local Deployment (coming soon)
To get started, clone this repository and run the following commands:
```
[coming soon]
```

### Cloud Dev Deployment
To get started, clone this repository and run the following commands:

Generate ssh keys for the DIMO node
```
ssh-keygen -t rsa -b 4096 -C "pulumi" -f keys/pulumi_key -N ""
```

#### Google Cloud
Authenticate to Google Cloud
```
gcloud auth login
gcloud config set project <project-name>
gcloud auth application-default login
```

CD to each stack directory [ infrastructure | dependencies | application ] and
do the following

Initialize Pulumi
```
# This will as you to name the stack, you can use the default <dev> name
pulumi stack init
```

Set the following Pulumi configuration variables:
(these will be set as defaults later)
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

Repeat in the next stack directory

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

# Troubleshooting & Debugging
## Pulumi
Running Pulumi in debug mode
```
pulumi up --logtostderr --logflow -v=10 2> out.txt
```

# Helpful Links
## Pulumi General
- [Pulumi](https://www.pulumi.com/docs/)
- [Pulumi GCP Provider](https://www.pulumi.com/docs/reference/pkg/gcp/)
- [Pulumi Organizational Paterns](https://www.pulumi.com/blog/organizational-patterns-infra-repo/)
## Pulumi Examples
- [EKS From Scratch](https://github.com/scottslowe/learning-tools/blob/main/pulumi/eks-from-scratch/vpc.go)
- [Pulumi Self Hosted Installers](https://github.com/pulumi/pulumi-self-hosted-installers/blob/master/ecs-hosted/go/infrastructure/main.go)
## Pulumi Golang
- [Using Go Generics in Pulumi](https://www.pulumi.com/blog/go-generics-preview/)