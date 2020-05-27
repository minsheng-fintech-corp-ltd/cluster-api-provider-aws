/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/scope"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/services/ec2"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/services/elb"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	infrav1 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
)

// TenantConfigReconciler reconciles a TenantConfig object
type TenantConfigReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=tenantconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=tenantconfigs/status,verbs=get;update;patch

func (r *TenantConfigReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	logger := r.Log.WithValues("tenantconfig", req.NamespacedName)
	logger.Info("reconciling tenant")
	tenantConfig := &infrav1.TenantConfig{}
	err := r.Get(ctx, req.NamespacedName, tenantConfig)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	awsCluster, err := GetAWSClusterByName(ctx, r.Client, tenantConfig.Namespace, tenantConfig.Spec.AWSClusterName)
	if err != nil || awsCluster == nil {
		if apierrors.IsNotFound(err) {
			logger.Info("awscluster is not defined")
		}
		logger.Info("failed to get owner awscluster")
		return reconcile.Result{}, err
	}

	cluster, err := util.GetOwnerCluster(ctx, r.Client, awsCluster.ObjectMeta)
	if err != nil || cluster == nil {
		logger.Info("failed to get owner cluster or owner reference is not set")
		return reconcile.Result{}, err
	}

	if util.IsPaused(cluster, tenantConfig) {
		logger.Info("TenantConfig or linked Cluster is marked as paused. Won't reconcile")
		return reconcile.Result{}, nil
	}

	logger = logger.WithValues("cluster", cluster.Name, "awsCluster", awsCluster.Name)

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		Logger:     logger,
		Cluster:    cluster,
		AWSCluster: awsCluster,
	})
	if err != nil {
		logger.Error(err, "failed to setup cluster scope")
		return reconcile.Result{}, err
	}

	tenantConfigScope, err := scope.NewTenantConfigScope(scope.TenantConfigParams{
		Logger:       logger,
		Client:       r.Client,
		TenantConfig: tenantConfig,
		ClusterScope: clusterScope,
	})
	if err != nil {
		logger.Error(err, "failed to setup tenant config scope")
		return reconcile.Result{}, err
	}

	// Always close the scope when exiting this function so we can persist any AWSCluster changes.
	defer func() {
		if err := tenantConfigScope.Close(); err != nil && reterr == nil {
			reterr = err
		}
	}()
	// Handle deleted clusters
	if !tenantConfig.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(tenantConfigScope)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(tenantConfigScope)
}

func (r *TenantConfigReconciler) reconcileDelete(tenantConfigScope *scope.TenantConfigScope) (reconcile.Result, error) {
	tenantConfigScope.Info("Reconciling TenantConfig delete")
	tenantConfig := tenantConfigScope.TenantConfig
	elbService := elb.NewService(tenantConfigScope)
	if err := elbService.DeleteLoadbalancers(); err != nil {
		tenantConfigScope.Logger.Error(err, "failed to delete load balancer")
		return ctrl.Result{}, errors.Wrapf(err, "error deleting load balancer for TenantConfig %s/%s", tenantConfig.Namespace, tenantConfig.Name)
	}
	ec2Service := ec2.NewService(tenantConfigScope)
	if err := ec2Service.DeleteNetwork(); err != nil {
		tenantConfigScope.Logger.Error(err, "failed to delete network")
		return ctrl.Result{}, errors.Wrapf(err, "error deleting network for TenantConfig %s/%s", tenantConfig.Namespace, tenantConfig.Name)
	}

	controllerutil.RemoveFinalizer(tenantConfig, infrav1.TenantConfigFinalizer)
	return reconcile.Result{}, nil
}

func (r *TenantConfigReconciler) reconcileNormal(tenantConfigScope *scope.TenantConfigScope) (reconcile.Result, error) {
	tenantConfigScope.Info("Reconciling AWSCluster")

	tenantConfig := tenantConfigScope.TenantConfig

	// If the AWSCluster doesn't have our finalizer, add it.
	controllerutil.AddFinalizer(tenantConfig, infrav1.TenantConfigFinalizer)
	// Register the finalizer immediately to avoid orphaning AWS resources on delete
	if err := tenantConfigScope.PatchObject(); err != nil {
		return reconcile.Result{}, err
	}

	ec2Service := ec2.NewService(tenantConfigScope)
	elbService := elb.NewService(tenantConfigScope)

	if err := ec2Service.ReconcileNetwork(); err != nil {
		return reconcile.Result{}, errors.Wrapf(err, "failed to reconcile network for AWSCluster %s/%s", tenantConfig.Namespace, tenantConfig.Name)
	}

	if tenantConfigScope.TenantConfig.Spec.ControlPlaneLoadBalancer != nil {
		if err := elbService.ReconcileLoadbalancers(); err != nil {
			return reconcile.Result{}, errors.Wrapf(err, "failed to reconcile load balancers for AWSCluster %s/%s", tenantConfig.Namespace, tenantConfig.Name)
		}
	}
	//todo: reconcile normal
	return reconcile.Result{}, nil
}

func (r *TenantConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.TenantConfig{}).
		Complete(r)
}
