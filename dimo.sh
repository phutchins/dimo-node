#!/bin/bash

GCP_PROJECT=dimo-dev-401815

echo "Creating infra stack...";
cd infrastructure
pulumi stack init dimo-dev --non-interactive
pulumi config set gcp:project ${GCP_PROJECT}
pulumi config set gcp:zone us-central1-a
pulumi config set gcp:region us-central1
pulumi install
# Different environments would be in different pulumi orgs
pulumi up -y
