package controllers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	infrav1 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetClusterByName finds and return a Cluster object using the specified params.
func GetAWSClusterByName(ctx context.Context, c client.Client, namespace, name string) (*infrav1.AWSCluster, error) {
	cluster := &infrav1.AWSCluster{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.Get(ctx, key, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// GetOwnerCluster returns the Cluster object owning the current resource.
func GetOwnerAWSClusterName(obj metav1.ObjectMeta) string {
	for _, ref := range obj.OwnerReferences {
		if ref.Kind == "AWSCluster" && ref.APIVersion == infrav1.GroupVersion.String() {
			return ref.Name
		}
	}
	return ""
}
