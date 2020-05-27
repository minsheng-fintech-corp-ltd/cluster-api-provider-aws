/*
Copyright 2020 The Kubernetes Authors.

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
	"testing"
)

func TestTenantConfig_ValidateUpdate(t *testing.T) {
	tests := []struct {
		name       string
		oldCluster *TenantConfig
		newCluster *TenantConfig
		wantErr    bool
	}{
		{
			name: "region is immutable",
			oldCluster: &TenantConfig{
				Spec: TenantConfigSpec{
					Region: "us-east-1",
				},
			},
			newCluster: &TenantConfig{
				Spec: TenantConfigSpec{
					Region: "us-east-2",
				},
			},
			wantErr: true,
		},
		{
			name: "controlPlaneLoadBalancer is immutable",
			oldCluster: &TenantConfig{
				Spec: TenantConfigSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Scheme: &ClassicELBSchemeInternal,
					},
				},
			},
			newCluster: &TenantConfig{
				Spec: TenantConfigSpec{
					ControlPlaneLoadBalancer: &AWSLoadBalancerSpec{
						Scheme: &ClassicELBSchemeInternetFacing,
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.newCluster.ValidateUpdate(tt.oldCluster); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
