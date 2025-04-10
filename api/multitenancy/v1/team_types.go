/*
Copyright 2024 Contributors to EdgeNet Project.

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

const TeamAdminRoleName = "edgenet:team-admin"

// TeamSpec defines the desired state of Team
type TeamSpec struct {
	// Full name of the team.
	// +kubebuilder:validation:MaxLength=80
	// +kubebuilder:validation:Required
	FullName string `json:"fullName"`

	// Description provides additional information about the team.
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:Optional
	Description string `json:"description"`

	// This is the admin username for the team. A role binding will be created for user with this username.
	// The username for some cases can also be emails. This was the old method. But with different identity
	// providers this can be any name.
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-.@_a-z0-9]*[a-z0-9])?$`
	Admin string `json:"admin"`

	// Website of the team.
	// +kubebuilder:validation:Pattern=`^(https?://)?([\da-z\.-]+)\.([a-z\.]{2,6})([/\w \.-]*)*/?$`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=2000
	URL string `json:"url"`

	// This represents the initial resource allocation for the team. If not specified, the team resource
	// quota will not be created.
	// +kubebuilder:validation:Required
	InitialRequest map[corev1.ResourceName]resource.Quantity `json:"initialRequest"`

	// Whether cluster-level network policies will be applied to team namespaces for security purposes.
	// +kubebuilder:default=false
	// +kubebuilder:validation:Optional
	ClusterNetworkPolicy bool `json:"clusterNetworkPolicy"`
}

// TeamStatus defines the observed state of Team
type TeamStatus struct {
	// The state can be Established or Failed.
	State string `json:"state"`

	// Additional description can be located here.
	Message string `json:"message"`

	// Failed sets the backoff limit.
	Failed int `json:"failed"`
}

// Team is the Schema for the teams API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=team
// +kubebuilder:printcolumn:name="Full Name",type="string",JSONPath=".spec.fullName"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.url"
// +kubebuilder:printcolumn:name="Admin",type="string",JSONPath=".spec.admin"
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}
