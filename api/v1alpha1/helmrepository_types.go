/*
Copyright 2025 The Ketches Authors.

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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HelmRepositorySpec defines the desired state of HelmRepository.
type HelmRepositorySpec struct {
	// URL is the repository URL
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^(https?|oci)://.*`
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`

	// Type specifies the repository type
	// +kubebuilder:validation:Enum=helm;oci
	// +kubebuilder:default=helm
	Type string `json:"type,omitempty"`

	// Interval specifies how often to sync the repository
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="30m"
	Interval string `json:"interval,omitempty"`

	// Auth contains authentication configuration
	// +optional
	Auth *RepositoryAuth `json:"auth,omitempty"`

	// Timeout specifies the timeout for repository operations
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="5m"
	Timeout string `json:"timeout,omitempty"`

	// Suspend tells the controller to suspend subsequent sync operations
	// +kubebuilder:default=false
	Suspend bool `json:"suspend,omitempty"`

	// ValuesConfigMapPolicy defines how chart values ConfigMaps are managed
	// +kubebuilder:validation:Enum=disabled;on-demand;lazy
	// +kubebuilder:default=disabled
	// +optional
	ValuesConfigMapPolicy string `json:"valuesConfigMapPolicy,omitempty"`

	// ValuesConfigMapRetention defines how long to keep ConfigMaps (e.g., "168h" for 7 days)
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="168h"
	// +optional
	ValuesConfigMapRetention string `json:"valuesConfigMapRetention,omitempty"`
}

// HelmRepositoryStatus defines the observed state of HelmRepository.
type HelmRepositoryStatus struct {
	// Conditions contains the different condition statuses for this repository
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// LastSyncTime is the last time the repository was successfully synced
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Charts contains the list of charts found in the repository
	// +optional
	Charts []ChartInfo `json:"charts,omitempty"`

	// Stats contains repository statistics
	// +optional
	Stats *RepositoryStats `json:"stats,omitempty"`

	// ObservedGeneration is the last generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// RepositoryAuth contains authentication configuration
type RepositoryAuth struct {
	// Basic contains basic authentication configuration
	// +optional
	Basic *BasicAuth `json:"basic,omitempty"`

	// TLS contains TLS configuration
	// +optional
	TLS *TLSConfig `json:"tls,omitempty"`
}

// BasicAuth contains basic authentication configuration
type BasicAuth struct {
	// Username for basic authentication
	// +optional
	Username string `json:"username,omitempty"`

	// Password for basic authentication
	// +optional
	Password string `json:"password,omitempty"`

	// SecretRef references a secret containing authentication credentials
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// TLSConfig contains TLS configuration
type TLSConfig struct {
	// InsecureSkipVerify controls whether to skip TLS certificate verification
	// +kubebuilder:default=false
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`

	// CAFile path to CA certificate file
	// +optional
	CAFile string `json:"caFile,omitempty"`

	// CertFile path to client certificate file
	// +optional
	CertFile string `json:"certFile,omitempty"`

	// KeyFile path to client private key file
	// +optional
	KeyFile string `json:"keyFile,omitempty"`

	// SecretRef references a secret containing TLS configuration
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// SecretReference contains reference to a secret
type SecretReference struct {
	// Name of the secret
	Name string `json:"name"`

	// Namespace of the secret
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// ChartInfo contains information about a chart
type ChartInfo struct {
	// Name of the chart
	Name string `json:"name"`

	// Description of the chart
	// +optional
	Description string `json:"description,omitempty"`

	// Versions contains the list of chart versions
	// +optional
	Versions []ChartVersion `json:"versions,omitempty"`
}

// ChartVersion contains information about a chart version
type ChartVersion struct {
	// Version of the chart
	Version string `json:"version"`

	// AppVersion of the application
	// +optional
	AppVersion string `json:"appVersion,omitempty"`

	// Created timestamp
	// +optional
	Created *metav1.Time `json:"created,omitempty"`

	// Digest of the chart
	// +optional
	Digest string `json:"digest,omitempty"`
}

// RepositoryStats contains repository statistics
type RepositoryStats struct {
	// TotalCharts is the total number of charts
	TotalCharts int `json:"totalCharts"`

	// TotalVersions is the total number of chart versions
	TotalVersions int `json:"totalVersions"`

	// LastIndexSize is the size of the last index file
	// +optional
	LastIndexSize string `json:"lastIndexSize,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.url`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// HelmRepository is the Schema for the helmrepositories API.
type HelmRepository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   HelmRepositorySpec   `json:"spec"`
	Status HelmRepositoryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HelmRepositoryList contains a list of HelmRepository.
type HelmRepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []HelmRepository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmRepository{}, &HelmRepositoryList{})
}
