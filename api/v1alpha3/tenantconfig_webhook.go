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

package v1alpha3

import (
	"fmt"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var tenantconfiglog = logf.Log.WithName("tenantconfig-resource")

func (r *TenantConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-infrastructure-cluster-x-k8s-io-v1alpha3-tenantconfig,mutating=false,failurePolicy=fail,groups=infrastructure.cluster.x-k8s.io,resources=tenantconfigs,versions=v1alpha3,name=vtenantconfig.kb.io

var _ webhook.Validator = &TenantConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *TenantConfig) ValidateCreate() error {
	tenantconfiglog.Info("validate create", "name", r.Name)
	var allErrs field.ErrorList

	if len(r.Spec.AWSClusterName) <= 0 {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "awsClusterName"), r.Spec.AWSClusterName, "field is null"),
		)
	}
	if len(r.Spec.RoleArn) <= 0 {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "roleArn"), r.Spec.RoleArn, "field is null"),
		)
	}

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *TenantConfig) ValidateUpdate(old runtime.Object) error {
	tenantconfiglog.Info("validate update", "name", r.Name)

	var allErrs field.ErrorList

	oldC, ok := old.(*TenantConfig)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected an TenantConfig but got a %T", old))
	}

	if r.Spec.Region != oldC.Spec.Region {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "region"), r.Spec.Region, "field is immutable"),
		)
	}

	if !reflect.DeepEqual(r.Spec.ControlPlaneLoadBalancer, oldC.Spec.ControlPlaneLoadBalancer) {
		allErrs = append(allErrs,
			field.Invalid(field.NewPath("spec", "controlPlaneLoadBalancer"),
				r.Spec.ControlPlaneLoadBalancer, "field is immutable"),
		)
	}

	return aggregateObjErrors(r.GroupVersionKind().GroupKind(), r.Name, allErrs)

}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *TenantConfig) ValidateDelete() error {
	tenantconfiglog.Info("validate delete", "name", r.Name)
	return nil
}
