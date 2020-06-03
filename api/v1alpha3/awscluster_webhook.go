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

package v1alpha3

import (
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var awsclusterlog = logf.Log.WithName("awscluster-resource")

func (r *AWSCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha3-awscluster,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=infrastructure.cluster.x-k8s.io,resources=awsclusters,versions=v1alpha3,name=validation.awscluster.infrastructure.cluster.x-k8s.io

var _ webhook.Validator = &AWSCluster{}

func (r *AWSCluster) ValidateCreate() error {
	awsclusterlog.Info("validate create", "name", r.Name, "namespace", r.Namespace)
	var allErrs field.ErrorList

	if len(r.Spec.TenantConfigName) <= 0 {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "tenantConfigName"), r.Spec.TenantConfigName, "field is not null"),
		)
	}

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

func (r *AWSCluster) ValidateDelete() error {
	return nil
}

func (r *AWSCluster) ValidateUpdate(old runtime.Object) error {
	oldC, ok := old.(*AWSCluster)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected an TenantConfig but got a %T", old))
	}

	var allErrs field.ErrorList

	if !reflect.DeepEqual(oldC.Spec.ControlPlaneEndpoint, clusterv1.APIEndpoint{}) &&
		!reflect.DeepEqual(r.Spec.ControlPlaneEndpoint, oldC.Spec.ControlPlaneEndpoint) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "controlPlaneEndpoint"), r.Spec.ControlPlaneEndpoint, "field is immutable"),
		)
	}
	if !reflect.DeepEqual(oldC.Spec.TenantConfigName, r.Spec.TenantConfigName) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "tenantConfigName"), r.Spec.TenantConfigName, "field is immutable"),
		)
	}
	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}
