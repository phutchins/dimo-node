import { interpolate, Config, Input, Output } from "@pulumi/pulumi";
import * as pulumi from "@pulumi/pulumi";
import * as gcp from "@pulumi/gcp";
import * as k8s from "@pulumi/kubernetes";
import { remote, types } from "@pulumi/command";
import * as fs from "fs";


// Import the program's configuration settings.
const config = new Config();
const machineType = config.get("machineType") || "f1-micro";
const osImage = config.get("osImage") || "debian-11";
const instanceTag = config.get("instanceTag") || "k3s";
const kubePort = config.get("kubePort") || "6443";
const credentialsPath = "./keys";
const publicKeyPath = `${credentialsPath}/id_rsa.pub`;
const privateKeyPath = `${credentialsPath}/id_rsa`;
const privateKey = fs.readFileSync(privateKeyPath, "utf-8");
const publicKey = fs.readFileSync(publicKeyPath, "utf-8");

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
                kubePort,
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

const instanceSSHKeys = new gcp.compute.ProjectMetadata("instanceSSHKeys", {metadata: {
    "ssh-keys": `pulumi:${publicKey}
                `,
}});

// Define a script to be run when the VM starts up.
// Can also use: curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="--tls-san x.x.x.x" sh -s -
const metadataStartupScript = `#!/bin/bash
    sudo apt-get update`;

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
    //metadataStartupScript,
    tags: [
        instanceTag,
    ],
}, { dependsOn: [ firewall, instanceSSHKeys ] });

export const instanceExtIP = instance.networkInterfaces.apply(interfaces => {
    return interfaces[0].accessConfigs![0].natIp;
});

export const instanceIntIP = instance.networkInterfaces.apply(interfaces => {
    return interfaces[0].networkIp;
});

export const instanceName = instance.name
// TODO: Set this to the correct port for k3s
export const kubeMaster = interpolate`${instanceIntIP}:${kubePort}`;

const sshUser: Input<string> = "pulumi";
const sshHost: Input<string> = instanceExtIP;

const k3sCommand = pulumi.all({instanceIntIP, instanceExtIP}).apply(({instanceIntIP, instanceExtIP}) => {
    console.log('Instance External IP is: ', instanceExtIP.toString());
    console.log('Instance Internal IP is: ', instanceIntIP.toString());

    return 'curl -sfL https://get.k3s.io | sh -s -- --bind-address ' + instanceIntIP.toString() + ' --tls-san ' + instanceExtIP.toString() + ' --advertise-address ' + instanceIntIP.toString() + ' --advertise-address ' + instanceIntIP.toString();
});

const getKubeConfigCmd = pulumi.all({instanceIntIP, instanceExtIP}).apply(({instanceIntIP, instanceExtIP}) => {
    return `sudo cat /etc/rancher/k3s/k3s.yaml | sed 's/.*server: .*/    server: https:\\/\\/${instanceExtIP}:6443/g'`;
});

const connection: types.input.remote.ConnectionArgs = {
    host: sshHost,
    user: sshUser,
    privateKey: privateKey,
};

function GetValue<T>(output: Output<T>) {
    return new Promise<T>((resolve, reject)=>{
        output.apply(value=>{
            resolve(value);
        });
    });
}

const installK3s = new remote.Command("install-k3s", {
    connection,
    create: k3sCommand,
}, { dependsOn: [ instance ] });

const fetchKubeconfig = new remote.Command("fetch-kubeconfig", {
    connection,
    create: getKubeConfigCmd,
}, { dependsOn: [ instance, installK3s ] });

const kubeConfig = fetchKubeconfig.stdout;

// Define a kubernetes provider instance that uses our cluster from above.
const kubeProvider = new k8s.Provider("k3s", {
    kubeconfig: kubeConfig,
}, { dependsOn: [ fetchKubeconfig ] });

// Create namespace for postgres-operator
const postgresOperatorNamespace = new k8s.core.v1.Namespace("postgres-operator", {
    metadata: {
        name: "postgres-operator",
    }},
    {
        provider: kubeProvider,
        dependsOn: [ kubeProvider ],
    },
);

// Deploy a helm chart to the cluster.
const pgOperator = new k8s.helm.v3.Chart("postgres-operator", {
    chart: "postgres-operator",
    fetchOpts: {
        repo: "https://opensource.zalando.com/postgres-operator/charts/postgres-operator",
    },
    namespace: "postgres-operator",
    values: {
        service: {
            type: "LoadBalancer",
        },
    },
}, { provider: kubeProvider });
