/*
Copyright 2019 The Kubernetes Authors.

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
	"net"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	infrav1 "sigs.k8s.io/cluster-api-provider-aws/api/v1alpha3"
	"sigs.k8s.io/cluster-api-provider-aws/pkg/cloud/scope"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/cluster-api/util"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// AWSClusterReconciler reconciles a AwsCluster object
type AWSClusterReconciler struct {
	client.Client
	Recorder record.EventRecorder
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=awsclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=awsclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=clusters;clusters/status,verbs=get;list;watch

func (r *AWSClusterReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.TODO()
	log := r.Log.WithValues("namespace", req.Namespace, "awsCluster", req.Name)

	// Fetch the AWSCluster instance
	awsCluster := &infrav1.AWSCluster{}
	err := r.Get(ctx, req.NamespacedName, awsCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Fetch the owner Cluster.
	cluster, err := util.GetOwnerCluster(ctx, r.Client, awsCluster.ObjectMeta)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		log.Info("Cluster Controller has not yet set OwnerRef")
		return ctrl.Result{}, nil
	}

	clusterScope, err := scope.NewClusterScope(scope.ClusterScopeParams{
		Client:     r.Client,
		Logger:     log,
		Cluster:    cluster,
		AWSCluster: awsCluster,
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		reterr = kerrors.NewAggregate([]error{reterr, clusterScope.Close()})
	}()

	// Handle deleted clusters
	if !awsCluster.DeletionTimestamp.IsZero() {
		// Cluster is deleted so remove the finalizer.
		//TODO: delete all tenantConfigs before cluster is delted.
		controllerutil.RemoveFinalizer(awsCluster, infrav1.ClusterFinalizer)
		return ctrl.Result{}, nil
	}

	log = log.WithValues("cluster", cluster.Name)

	// ensure tenant config
	tenantConfig := &infrav1.TenantConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      awsCluster.Spec.TenantConfigName,
			Namespace: awsCluster.Namespace,
		},
	}

	key, err := client.ObjectKeyFromObject(tenantConfig)
	if err != nil {
		log.Error(err, "failed to create tenant config reference")
		return reconcile.Result{}, errors.Errorf("failed to create tenant config reference: %+v", err)
	}
	if err := r.Client.Get(ctx, key, tenantConfig); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "failed to get tenant config")
			return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, errors.Errorf("failed to get tenant config:%+v", err)
		}
	}

	if !tenantConfig.Status.Ready {
		log.Info("wait until tenant config is ready")
		return reconcile.Result{}, nil
	}

	if tenantConfig.Status.APIServerELB.DNSName == "" {
		log.Info("Waiting on API server ELB DNS name")
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	if _, err := net.LookupIP(tenantConfig.Status.APIServerELB.DNSName); err != nil {
		log.Info("Waiting on API server ELB DNS name to resolve")
		return reconcile.Result{RequeueAfter: 15 * time.Second}, nil
	}

	awsCluster.Spec.ControlPlaneEndpoint = clusterv1.APIEndpoint{
		Host: tenantConfig.Status.APIServerELB.DNSName,
		Port: clusterScope.APIServerPort(),
	}
	awsCluster.Status.Ready = true
	return reconcile.Result{}, nil
}

func (r *AWSClusterReconciler) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	controller, err := ctrl.NewControllerManagedBy(mgr).
		WithOptions(options).
		For(&infrav1.AWSCluster{}).
		WithEventFilter(pausedPredicates(r.Log)).
		Owns(&infrav1.TenantConfig{}).
		Build(r)

	if err != nil {
		return errors.Wrap(err, "error creating controller")
	}

	return controller.Watch(
		&source.Kind{Type: &clusterv1.Cluster{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: handler.ToRequestsFunc(r.requeueAWSClusterForUnpausedCluster),
		},
		predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldCluster := e.ObjectOld.(*clusterv1.Cluster)
				newCluster := e.ObjectNew.(*clusterv1.Cluster)
				log := r.Log.WithValues("predicate", "updateEvent", "namespace", newCluster.Namespace, "cluster", newCluster.Name)
				switch {
				// return true if Cluster.Spec.Paused has changed from true to false
				case oldCluster.Spec.Paused && !newCluster.Spec.Paused:
					log.V(4).Info("Cluster was unpaused, will attempt to map associated AWSCluster.")
					return true
				// otherwise, return false
				default:
					log.V(4).Info("Cluster did not match expected conditions, will not attempt to map associated AWSCluster.")
					return false
				}
			},
			CreateFunc: func(e event.CreateEvent) bool {
				cluster := e.Object.(*clusterv1.Cluster)
				log := r.Log.WithValues("predicate", "createEvent", "namespace", cluster.Namespace, "cluster", cluster.Name)

				// Only need to trigger a reconcile if the Cluster.Spec.Paused is false
				if !cluster.Spec.Paused {
					log.V(4).Info("Cluster is not paused, will attempt to map associated AWSCluster.")
					return true
				}
				log.V(4).Info("Cluster did not match expected conditions, will not attempt to map associated AWSCluster.")
				return false
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				log := r.Log.WithValues("predicate", "deleteEvent", "namespace", e.Meta.GetNamespace(), "cluster", e.Meta.GetName())
				log.V(4).Info("Cluster did not match expected conditions, will not attempt to map associated AWSCluster.")
				return false
			},
			GenericFunc: func(e event.GenericEvent) bool {
				log := r.Log.WithValues("predicate", "genericEvent", "namespace", e.Meta.GetNamespace(), "cluster", e.Meta.GetName())
				log.V(4).Info("Cluster did not match expected conditions, will not attempt to map associated AWSCluster.")
				return false
			},
		},
	)
}

func (r *AWSClusterReconciler) requeueAWSClusterForUnpausedCluster(o handler.MapObject) []ctrl.Request {
	c := o.Object.(*clusterv1.Cluster)
	log := r.Log.WithValues("objectMapper", "clusterToAWSCluster", "namespace", c.Namespace, "cluster", c.Name)

	// Don't handle deleted clusters
	if !c.ObjectMeta.DeletionTimestamp.IsZero() {
		log.V(4).Info("Cluster has a deletion timestamp, skipping mapping.")
		return nil
	}

	// Make sure the ref is set
	if c.Spec.InfrastructureRef == nil {
		log.V(4).Info("Cluster does not have an InfrastructureRef, skipping mapping.")
		return nil
	}

	if c.Spec.InfrastructureRef.GroupVersionKind().Kind != "AWSCluster" {
		log.V(4).Info("Cluster has an InfrastructureRef for a different type, skipping mapping.")
		return nil
	}

	log.V(4).Info("Adding request.", "awsCluster", c.Spec.InfrastructureRef.Name)
	return []ctrl.Request{
		{
			NamespacedName: client.ObjectKey{Namespace: c.Namespace, Name: c.Spec.InfrastructureRef.Name},
		},
	}
}
