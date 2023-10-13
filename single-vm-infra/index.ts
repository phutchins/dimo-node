import * as pulumi from "@pulumi/pulumi";
import * as gcp from "@pulumi/gcp";
import * as k8s from "@pulumi/kubernetes";

// Import the program's configuration settings.
const config = new pulumi.Config();
const machineType = config.get("machineType") || "f1-micro";
const osImage = config.get("osImage") || "debian-11";
const instanceTag = config.get("instanceTag") || "webserver";
const servicePort = config.get("servicePort") || "80";

// Create a new network for the virtual machine.
const network = new gcp.compute.Network("network", {
    autoCreateSubnetworks: false,
});

// Create a subnet on the network.
const subnet = new gcp.compute.Subnetwork("subnet", {
    ipCidrRange: "10.0.1.0/24",
    network: network.id,
});

// Create a firewall allowing inbound access over ports 80 (for HTTP) and 22 (for SSH).
const firewall = new gcp.compute.Firewall("firewall", {
    network: network.selfLink,
    allows: [
        {
            protocol: "tcp",
            ports: [
                "22",
                servicePort,
            ],
        },
    ],
    direction: "INGRESS",
    sourceRanges: [
        "0.0.0.0/0",
    ],
    targetTags: [
        instanceTag,
    ],
});

const mySshKey = new gcp.compute.ProjectMetadata("mySshKey", {metadata: {
  "ssh-keys": `      philip:ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCYkAVnwBSnzYv+kDykNrAD3CxFSFUo5lXNGJecUMFs4VKva0DpNQ9wAjI5EhapTkr8tW0faD3iCZRR3w4/c/+jxVTJ3fiLBKjL3AAujsBeQV7m9n6cWxlKuZlgUFF7B50by9aHXaLOTAM6jqEdtVAau0WSYiz5IoDt8GgMw9k2mJBKWv6vWaQ9sfU9LHOym9AAS5ksPhV4q26Fy4J9IoTasGoXcaJQw+wojtqm4Ws3lAA5bhxnTrkxRH38MHHY0UQU2lj5MAisB4lQMWn0gxZtHpc+tlTEte226jEq+b48LSITcgjl/tD1eWYVZxnWWcffQijnTOB4hpvDDOGVV1Rn philip
  `,
}});

// Define a script to be run when the VM starts up.
const metadataStartupScript = `#!/bin/bash
                               sudo apt-get update
                               curl -sfL https://get.k3s.io | sh -`;


// Create the virtual machine.
const instance = new gcp.compute.Instance("instance", {
    machineType,
    bootDisk: {
        initializeParams: {
            image: osImage,
        },
    },
    networkInterfaces: [
        {
            network: network.id,
            subnetwork: subnet.id,
            accessConfigs: [
                {},
            ],
        },
    ],
    serviceAccount: {
        scopes: [
            "https://www.googleapis.com/auth/cloud-platform",
        ],
    },
    allowStoppingForUpdate: true,
    metadataStartupScript,
    tags: [
        instanceTag,
    ],
}, { dependsOn: firewall });

const instanceIP = instance.networkInterfaces.apply(interfaces => {
    return interfaces[0].accessConfigs![0].natIp;
});

// Export the instance's name, public IP address, and HTTP URL.
export const name = instance.name;
export const ip = instanceIP;
export const url = pulumi.interpolate`http://${instanceIP}:${servicePort}`;

// Deploy a helm chart to the cluster.
const pgOperator = new k8s.helm.v3.Chart("postgres-operator", {
    chart: "postgres-operator",
    fetchOpts: {
        repo: "https://opensource.zalando.com/postgres-operator/charts/postgres-operator",
    },
    namespace: "default",
    values: {
        service: {
            type: "LoadBalancer",
        },
    },
}, { provider: clusterProvider });
