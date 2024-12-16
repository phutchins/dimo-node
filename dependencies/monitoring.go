package dependencies

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	//apiextensions "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/apiextensions/v1"
	//metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func InstallMonitoringDependencies(ctx *pulumi.Context, kubeProvider *kubernetes.Provider) (err error) {
	// err = utils.CreateNamespaces(ctx, kubeProvider, []string{"monitoring"})
	// if err != nil {
	// 	return err
	// }

	// _, err = helm.NewRelease(ctx, "kube-prometheus-stack-crds", &helm.ReleaseArgs{
	// 	Chart: pulumi.String("./dependencies/charts/kube-prometheus-stack/charts/crds"),
	// 	/*
	// 		ValueYamlFiles: pulumi.AssetOrArchiveArray{
	// 			pulumi.NewFileAsset("./dependencies/charts/kube-prometheus-stack/charts/crds/values.yaml"),
	// 		}, */
	// 	Namespace: pulumi.String("monitoring"),
	// }, pulumi.Provider(kubeProvider))
	// if err != nil {
	// 	return err
	// }

	//
	//// Create the Prometheus Operator CRD
	//_, err = apiextensions.NewCustomResourceDefinition(ctx, "prometheus-operator", &apiextensions.CustomResourceDefinitionArgs{
	//	Metadata: &metav1.ObjectMetaArgs{
	//		Name: pulumi.String("prometheus-operator"),
	//	},
	//	Spec: &apiextensions.CustomResourceDefinitionSpecArgs{
	//		Group: pulumi.String("monitoring.coreos.com"),
	//		Versions: apiextensions.CustomResourceDefinitionVersionArray{
	//			&apiextensions.CustomResourceDefinitionVersionArgs{
	//				Name:    pulumi.String("v1"),
	//				Served:  pulumi.Bool(true),
	//				Storage: pulumi.Bool(true),
	//			},
	//		},
	//		Scope: pulumi.String("Namespaced"),
	//		Names: &apiextensions.CustomResourceDefinitionNamesArgs{
	//			Plural: pulumi.String("prometheuses"),
	//			Kind:   pulumi.String("Prometheus"),
	//		},
	//	},
	//})
	//if err != nil {
	//	return err
	//}

	return nil
}
