/*
Copyright 2024 EdgeNet.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// Full name of the tenant.
	// +kubebuilder:validation:MaxLength=80
	// +kubebuilder:validation:Required
	FullName string `json:"fullName"`

	// Description provides additional information about the tenant.
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:Optional
	Description string `json:"description"`

	// Email provides a contact email for the tenant.
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,6}$`
	Email string `json:"email"`

	// Website of the tenant.
	// +kubebuilder:validation:Pattern=`^(https?://)?([\da-z\.-]+)\.([a-z\.]{2,6})([/\w \.-]*)*/?$`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=2000
	URL string `json:"url"`

	// +kubebuilder:validation:Optional
	ResourceAllocation map[corev1.ResourceName]resource.Quantity `json:"resourceAllocation"`

	// Whether cluster-level network policies will be applied to tenant namespaces for security purposes.
	// +kubebuilder:default=false
	// +kubebuilder:validation:Optional
	ClusterNetworkPolicy bool `json:"clusterNetworkPolicy"`

	// If the tenant is active then this field is true.
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`
}

// TenantStatus defines the observed state of Tenant
type TenantStatus struct {
	// The state can be Established or Failed.
	State string `json:"state"`

	// Additional description can be located here.
	Message string `json:"message"`

	// Failed sets the backoff limit.
	Failed int `json:"failed"`
}

// Tenant is the Schema for the tenants API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Official Name",type="string",JSONPath=".spec.fullname"
// +kubebuilder:printcolumn:name="Short Name",type="string",JSONPath=".spec.shortname"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.url"
// +kubebuilder:printcolumn:name="Email",type="string",JSONPath=".spec.email"
// +kubebuilder:printcolumn:name="Enabled",type="boolean",JSONPath=".spec.enabled"
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TenantList contains a list of Tenant
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}
