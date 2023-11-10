import { interpolate, Config, Input, Output } from "@pulumi/pulumi";
import * as pulumi from "@pulumi/pulumi";
import * as gcp from "@pulumi/gcp";
import * as k8s from "@pulumi/kubernetes";
import { remote, types } from "@pulumi/command";
import * as resource from "@pulumi/pulumi/dynamic";
import * as fs from "fs";
import * as k8s from "@kubernetes/client-node";

// TODO
// - Break out the k3s portion of the provider into its own file
// - Create a k8s provider in a file that creates a GKE cluster
// - Create a k8s provider in a file that creates a AKS cluster


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


/*
interface NodeListResourceProviderInputs {
    id: string;
    kubeConfig: pulumi.Input<any>;
}

interface NodeListResourceProviderOutputs {
    //nodeNames: pulumi.Output<string[]>;
    nodeNames: pulumi.provider.CreateResult;
}

class NodeListResourceProvider implements pulumi.dynamic.ResourceProvider {
    async create(inputs: NodeListResourceProviderInputs): Promise<pulumi.dynamic.CreateResult> {
        //async create(id: string, props: pulumi.Output<Record<string, unknown>>): Promise<resource.ReadResult> {
        const k8sProvider = new k8s.Provider(inputs.id, { kubeConfig });
        const nodeList = await k8s.core.v1.NodeList.get(id, undefined, { provider: k8sProvider });
        const nodeNames: pulumi.provider.CreateResult = nodeList.items.map(node => node.metadata?.name || "");
        
        return {
            nodeNames
        };
    }
}

class NodeListResource extends pulumi.dynamic.Resource {
    public readonly nodeNames!: pulumi.Output<string[]>;
  
    constructor(name: string, kubeconfig: pulumi.Input<any>, opts?: pulumi.CustomResourceOptions) {
      super(new pulumi.dynamic.ResourceProvider("node-list-provider", {

      }), name, { nodeNames: [] }, opts);
    }
  }
}
  */

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
                "31544",
            ],
        },
    ],
    direction: "INGRESS",
    sourceRanges: [
        "24.30.56.126/32",
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
    sudo apt-get update && sudo apt install -y jq`;

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

// Public IP
const publicIP = new gcp.compute.Address("public-ip", {
    region: "us-central1",
    addressType: "EXTERNAL",
    //address: instanceExtIP,
}, { dependsOn: [ instance ] });

const k3sCommand = pulumi.all({instanceIntIP, instanceExtIP}).apply(({instanceIntIP, instanceExtIP}) => {
    console.log('Instance External IP is: ', instanceExtIP.toString());
    console.log('Instance Internal IP is: ', instanceIntIP.toString());

    return 'curl -sfL https://get.k3s.io | sh -s -- --bind-address ' + instanceIntIP.toString() + ' --tls-san ' + instanceExtIP.toString() + ' --advertise-address ' + instanceIntIP.toString() + ' --advertise-address ' + instanceIntIP.toString() + ' --disable servicelb --write-kubeconfig-mode=644';
});

const getKubeConfigCmd = pulumi.all({instanceIntIP, instanceExtIP}).apply(({instanceIntIP, instanceExtIP}) => {
    return `sudo cat /etc/rancher/k3s/k3s.yaml | sed 's/.*server: .*/    server: https:\\/\\/${instanceExtIP}:6443/g'`;
});

const getKubeNodesCmd = pulumi.all({instanceIntIP, instanceExtIP}).apply(({instanceIntIP, instanceExtIP}) => {
    return `sudo kubectl get nodes -o json | jq '[.items[].metadata.name]'`;
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









export const fetchKubeNodes = new remote.Command("fetch-kube-nodes", {
    connection,
    create: getKubeNodesCmd,
}, { dependsOn: [ instance, installK3s ] });


export const kubeNodes = pulumi.all({fetchKubeNodes}).apply(({fetchKubeNodes}) => {
    console.log('Kube nodes: ', fetchKubeNodes.stdout)
    console.log('kube nodes JSON: ', fetchKubeNodes.stdout.apply(JSON.stringify))
    return fetchKubeNodes.stdout.apply(JSON.stringify);
});

// Define a kubernetes provider instance that uses our cluster from above.
const kubeProvider = new k8s.Provider("k3s", {
    kubeconfig: kubeConfig,
}, { dependsOn: [ fetchKubeconfig ] });



// TODO: Figure out how to get a client
//const kubeConfig = new k8s.KubeConfig();
//kubeConfig.loadFromDefault();

const myKubeConfig = kubeProvider.kubeconfig;


const k8sApi = myKubeConfig.makeApiClient(k8s.CoreV1Api);

k8sApi.listNode().then((res) => {
    console.log(res.body);
});




//const nodeList = new NodeListResource("node-list", kubeConfig);

//export const nodeNames = nodeList.nodeNames;

//let nodes: k8s.core.v1.NodeList;

export const nodes = pulumi.all({kubeProvider}).apply(({kubeProvider}) => {
    return k8s.core.v1.NodeList.get("nodes", kubeProvider.id, { provider: kubeProvider });
});

//let nodes = k8s.core.v1.NodeList.get("nodes", "k3s", { provider: kubeProvider, dependsOn: [ kubeProvider ] });
//export const nodeNames = nodes.items;
//nodes.items.then((items: { name: any; }[]) => items?.forEach((item: { name: any; }) => console.log(item.name)));

// Array of node names from k3s cluster
//const nodeNames = new k8s.core.v1.Node("nodes", {}, { provider: kubeProvider }).metadata.apply(m => m.map(n => n.name));

// Create namespace for metallb
const metallbNamespace = new k8s.core.v1.Namespace("metallb-system", {
    metadata: {
        name: "metallb-system",
    }},
    {
        provider: kubeProvider,
        dependsOn: [ kubeProvider ],
    },
);

// Install MetalLB with Helm
const metallb = new k8s.helm.v3.Chart("metallb", {
    chart: "metallb",
    fetchOpts: {
        repo: "https://metallb.github.io/metallb",
    },
    namespace: "metallb-system",
    values: {
    },
}, { provider: kubeProvider });

// Create namespace for nginx-ingress
const nginxIngressNamespace = new k8s.core.v1.Namespace("nginx-ingress", {
    metadata: {
        name: "nginx-ingress",
    }},
    {
        provider: kubeProvider,
        dependsOn: [ kubeProvider ],
    },
);

const nginxIngress = new k8s.helm.v3.Chart("nginx-ingress", {
    chart: "nginx-ingress",
    fetchOpts: {
        repo: "https://helm.nginx.com/stable",
    },
    namespace: "nginx-ingress",
    values: {
        controller: {
            publishService: {
                enabled: true,
            },
            service: {
                //type: "LoadBalancer",
                loadBalanderIP: publicIP.address
            },
        },
    },
}, { provider: kubeProvider,
    dependsOn: [ kubeProvider, nginxIngressNamespace]
});

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

// Create namespace for postgres
const postgresNamespace = new k8s.core.v1.Namespace("postgres", {
    metadata: {
        name: "postgres",
    }},
    {
        provider: kubeProvider,
        dependsOn: [ kubeProvider ],
    },
);

// Create namespace for kafka-operator
const kafkaNamespace = new k8s.core.v1.Namespace("kafka", {
    metadata: {
        name: "kafka",
    }},
    {
        provider: kubeProvider,
        dependsOn: [ kubeProvider ],
    },
);

// Create namespace for devices-api
const devicesApiNamespace = new k8s.core.v1.Namespace("devices-api", {
    metadata: {
        name: "devices-api",
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
            type: "ClusterIP",
        },
    },
}, { provider: kubeProvider });

// Deploy a helm chart to the cluster.
const pgOperatorUI = new k8s.helm.v3.Chart("postgres-operator-ui", {
    chart: "postgres-operator-ui",
    fetchOpts: {
        repo: "https://opensource.zalando.com/postgres-operator/charts/postgres-operator-ui",
    },
    namespace: "postgres-operator",
    values: {
        service: {
            type: "ClusterIP",
        },
    },
}, { provider: kubeProvider });

// Docs: https://staging.artifacthub.io/packages/helm/k8s-at-home/zalando-postgres-cluster
// Todo: 
// - Configure backups
// - Link to password rotation
const pgCluster = new k8s.helm.v3.Chart("zalando-postgres-cluster", {
    chart: "zalando-postgres-cluster",
    path: "./charts/",
    /*
    fetchOpts: {
        path: "./charts/zalando-postgres-cluster",
        // Or k8s-at-home/zalando-postgres-cluster ?
        // RE: https://staging.artifacthub.io/packages/helm/k8s-at-home/zalando-postgres-cluster
    }, */
    namespace: "postgres",
    values: {
        superuser: {
            user: "dimo_admin",
            password: "dimo_default_password",
            secret: "credentials.postgresql.acid.zalan.do",
        },
        postgresql: {
            postgresql: {
                version: "13",
            },
            users: {
                "dimo_admin": ["superuser", "createdb"],
            },
            databases: {
                "dimo": "dimo_admin"
            },
            volume: {
                size: "1Gi",
                storageClass: "standard", // Name of the storage class to use.
            }
        },
        persistentVolumes: {
            accessModes: ["ReadWriteOnce"],
            //replicaNodes: kubeNodes,
            //replicaNodes: 1,
            replicaNodes: [ "instance-10e4e90"],
                //"instance-10e4e90", // TODO: Make this dynamic and get the name of nodes from the cluster
            hostPathPrefix: "/mnt/data",
        },
        setup: {
        }
    },
}, { provider: kubeProvider,
    dependsOn: [ pgOperator, pgOperatorUI, postgresOperatorNamespace ] });

// Install Kafka with Helm
// Pass bootstrap URL to the app
const kafka = new k8s.helm.v3.Chart("kafka", {
    chart: "kafka",
    fetchOpts: {
        repo: "https://charts.bitnami.com/bitnami",
    },
    namespace: "kafka",
    values: {
        service: {
            type: "ClusterIP",
        },
        global: {
            storageClass: "standard",
        },
    },
}, { provider: kubeProvider,
    dependsOn: [ kafkaNamespace ] });

const devicesApi = new k8s.helm.v3.Chart("identity-api", {
    chart: "identity-api",
    path: "./dimo-apps/identity-api/charts",
    namespace: "identity-api",
}, { provider: kubeProvider,
    dependsOn: [ pgCluster, kafka ] });