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
├── applications
│   ├── dimo-identity
├── node
├── scripts
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

### Local Deployment
To get started, clone this repository and run the following commands:
```
asdf install
asdf reshim
```

### Cloud Deployment
To get started, clone this repository and run the following commands:
```
asdf install
asdf reshim
```

## Deployment Management Scripts

