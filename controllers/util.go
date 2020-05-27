package controllers

import (
	"context"

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
