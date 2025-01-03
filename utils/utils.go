package utils

import (
	"fmt"

	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ConvertString(s *string) pulumi.String {
	return pulumi.String(fmt.Sprintf("%v", s))
}

func ToPulumiStringArray(a []string) pulumi.StringArrayInput {
	var res []pulumi.StringInput
	for _, s := range a {
		res = append(res, pulumi.String(s))
	}
	return pulumi.StringArray(res)
}

func CreateNamespaces(ctx *pulumi.Context, kubeProvider *kubernetes.Provider, namespaces []string) (map[string]*corev1.Namespace, error) {
	namespaceMap := make(map[string]*corev1.Namespace)
	for _, namespace := range namespaces {
		ns, err := corev1.NewNamespace(ctx, fmt.Sprintf("%s", namespace), &corev1.NamespaceArgs{
			Metadata: &metav1.ObjectMetaArgs{
				Name: pulumi.String(namespace),
			},
		}, pulumi.Provider(kubeProvider))
		if err != nil {
			return nil, err
		}
		namespaceMap[namespace] = ns
	}

	return namespaceMap, nil
}
