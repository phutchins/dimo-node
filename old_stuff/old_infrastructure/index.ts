import * as pulumi from '@pulumi/pulumi';
import * as gcp from '@pulumi/gcp';
import * as k8s from '@pulumi/kubernetes';

// Create a GCP instance
const instance = new gcp.compute.Instance('instance', {
    machineType: 'e2-medium',
    bootDisk: {
        initializeParams: {
            image: 'projects/debian-cloud/global/images/debian-10-buster-v20210721', // Update the Debian image used
        },
    },
    networkInterfaces: [{
        network: 'default', // Specify the VPC network
        accessConfigs: [{}],
    }],
    zone: 'us-central1-a',
    metadataStartupScript: `#!/bin/bash
                            sudo apt-get update
                            curl -sfL https://get.k3s.io | sh -`,
});

// Export the VM instance name
export const instanceName = instance.name;
export const externalIp = instance.networkInterfaces.apply(ni => ni[0].accessConfigs[0].natIp);

// Authenticate to the k3s cluster
const k8sProvider = new k8s.Provider("k3sProvider", {
    host: pulumi.interpolate`https://${instance.networkInterfaces[0].accessConfigs[0].natIp}`,
    username: "admin",
    password: instance.metadata["kube-admin-password"],
    clientCertificate: instance.metadata["kube-admin-client-cert"],
    clientKey: instance.metadata["kube-admin-client-key"],
    clusterCaCertificate: instance.metadata["kube-ca-cert"],
    kubeconfig: pulumi.interpolate`${instance.metadata}`,
});

// Export the kubeconfig
export const kubeconfig = k8sProvider.kubeconfig;
