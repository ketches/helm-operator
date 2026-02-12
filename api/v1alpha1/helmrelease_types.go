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

// HelmReleaseSpec defines the desired state of HelmRelease.
type HelmReleaseSpec struct {
	// Chart specifies the chart information
	// +kubebuilder:validation:Required
	Chart ChartSpec `json:"chart"`

	// Release contains release configuration
	// +optional
	Release *ReleaseSpec `json:"release,omitempty"`

	// Values contains custom values for the chart as YAML string
	// +optional
	Values string `json:"values,omitempty"`

	// Install contains installation configuration
	// +optional
	Install *InstallSpec `json:"install,omitempty"`

	// Upgrade contains upgrade configuration
	// +optional
	Upgrade *UpgradeSpec `json:"upgrade,omitempty"`

	// Uninstall contains uninstallation configuration
	// +optional
	Uninstall *UninstallSpec `json:"uninstall,omitempty"`

	// Interval specifies how often to reconcile the release
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +optional
	Interval string `json:"interval,omitempty"`

	// Suspend tells the controller to suspend subsequent reconciliations
	// +kubebuilder:default=false
	Suspend bool `json:"suspend,omitempty"`

	// DependsOn contains references to other releases this release depends on
	// +optional
	DependsOn []DependencyReference `json:"dependsOn,omitempty"`

	// Rollback contains rollback configuration
	// +optional
	Rollback *RollbackSpec `json:"rollback,omitempty"`
}

// HelmReleaseStatus defines the observed state of HelmRelease.
type HelmReleaseStatus struct {
	// Conditions contains the different condition statuses for this release
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// HelmRelease contains information about the Helm release
	// +optional
	HelmRelease *HelmReleaseInfo `json:"helmRelease,omitempty"`

	// LastAppliedConfiguration contains the last applied configuration
	// +optional
	LastAppliedConfiguration *HelmReleaseSpec `json:"lastAppliedConfiguration,omitempty"`

	// OriginalValues contains the default values from the chart for comparison
	// +optional
	OriginalValues string `json:"originalValues,omitempty"`

	// Failures contains information about failed operations
	// +optional
	Failures []FailureRecord `json:"failures,omitempty"`

	// ObservedGeneration is the last generation observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ChartSpec specifies chart information
type ChartSpec struct {
	// Name of the chart
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Version of the chart
	// +optional
	Version string `json:"version,omitempty"`

	// Repository contains repository reference
	// +optional
	Repository *RepositoryReference `json:"repository,omitempty"`

	// RepositoryURL is the direct URL to the repository
	// +optional
	RepositoryURL string `json:"repositoryURL,omitempty"`

	// OCIRepository is the OCI registry URL for the chart (e.g., oci://registry.example.com/charts/mychart)
	// +optional
	OCIRepository string `json:"ociRepository,omitempty"`
}

// RepositoryReference contains reference to a HelmRepository
type RepositoryReference struct {
	// Name of the HelmRepository
	Name string `json:"name"`

	// Namespace of the HelmRepository
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// ReleaseSpec contains release configuration
type ReleaseSpec struct {
	// Name of the release
	// +optional
	Name string `json:"name,omitempty"`

	// Namespace where the release will be installed
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// CreateNamespace indicates whether to create the namespace if it doesn't exist
	// +kubebuilder:default=false
	CreateNamespace bool `json:"createNamespace,omitempty"`
}

// InstallSpec contains installation configuration
type InstallSpec struct {
	// Timeout for the install operation
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="10m"
	Timeout string `json:"timeout,omitempty"`

	// Wait indicates whether to wait for the installation to complete
	// +kubebuilder:default=true
	Wait bool `json:"wait,omitempty"`

	// WaitForJobs indicates whether to wait for jobs to complete
	// +kubebuilder:default=true
	WaitForJobs bool `json:"waitForJobs,omitempty"`

	// SkipCRDs indicates whether to skip CRD installation
	// +kubebuilder:default=false
	SkipCRDs bool `json:"skipCRDs,omitempty"`

	// Replace indicates whether to replace existing resources
	// +kubebuilder:default=false
	Replace bool `json:"replace,omitempty"`

	// DisableHooks indicates whether to disable hooks
	// +kubebuilder:default=false
	DisableHooks bool `json:"disableHooks,omitempty"`
}

// UpgradeSpec contains upgrade configuration
type UpgradeSpec struct {
	// Timeout for the upgrade operation
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="10m"
	Timeout string `json:"timeout,omitempty"`

	// Wait indicates whether to wait for the upgrade to complete
	// +kubebuilder:default=true
	Wait bool `json:"wait,omitempty"`

	// WaitForJobs indicates whether to wait for jobs to complete
	// +kubebuilder:default=true
	WaitForJobs bool `json:"waitForJobs,omitempty"`

	// Force indicates whether to force upgrade
	// +kubebuilder:default=false
	Force bool `json:"force,omitempty"`

	// ResetValues indicates whether to reset values to chart defaults
	// +kubebuilder:default=false
	ResetValues bool `json:"resetValues,omitempty"`

	// ReuseValues indicates whether to reuse existing values
	// +kubebuilder:default=false
	ReuseValues bool `json:"reuseValues,omitempty"`

	// Recreate indicates whether to recreate resources
	// +kubebuilder:default=false
	Recreate bool `json:"recreate,omitempty"`

	// MaxHistory limits the maximum number of revisions saved per release
	// +kubebuilder:default=10
	MaxHistory int `json:"maxHistory,omitempty"`

	// CleanupOnFail indicates whether to cleanup on failure
	// +kubebuilder:default=true
	CleanupOnFail bool `json:"cleanupOnFail,omitempty"`

	// DisableHooks indicates whether to disable hooks
	// +kubebuilder:default=false
	DisableHooks bool `json:"disableHooks,omitempty"`
}

// UninstallSpec contains uninstallation configuration
type UninstallSpec struct {
	// Timeout for the uninstall operation
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="5m"
	Timeout string `json:"timeout,omitempty"`

	// DisableHooks indicates whether to disable hooks
	// +kubebuilder:default=false
	DisableHooks bool `json:"disableHooks,omitempty"`

	// KeepHistory indicates whether to keep release history
	// +kubebuilder:default=false
	KeepHistory bool `json:"keepHistory,omitempty"`
}

// RollbackSpec contains rollback configuration
type RollbackSpec struct {
	// Enabled enables automatic rollback on upgrade failure
	// +kubebuilder:default=false
	Enabled bool `json:"enabled,omitempty"`

	// ToRevision specifies the revision to rollback to (0 means previous revision)
	// +kubebuilder:default=0
	// +optional
	ToRevision int `json:"toRevision,omitempty"`

	// Timeout for the rollback operation
	// +kubebuilder:validation:Pattern=`^([0-9]+(\.[0-9]+)?(ms|s|m|h))+$`
	// +kubebuilder:default="5m"
	// +optional
	Timeout string `json:"timeout,omitempty"`

	// Wait indicates whether to wait for rollback to complete
	// +kubebuilder:default=true
	Wait bool `json:"wait,omitempty"`

	// CleanupOnFail indicates whether to cleanup failed rollback
	// +kubebuilder:default=true
	CleanupOnFail bool `json:"cleanupOnFail,omitempty"`

	// Force indicates whether to force rollback through deletion
	// +kubebuilder:default=false
	Force bool `json:"force,omitempty"`

	// DisableHooks indicates whether to disable hooks during rollback
	// +kubebuilder:default=false
	DisableHooks bool `json:"disableHooks,omitempty"`
}

// DependencyReference contains reference to a dependency
type DependencyReference struct {
	// Name of the dependency release
	Name string `json:"name"`

	// Namespace of the dependency release
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// HelmReleaseInfo contains information about a Helm release
type HelmReleaseInfo struct {
	// Name of the release
	Name string `json:"name"`

	// Namespace of the release
	Namespace string `json:"namespace"`

	// Revision of the release
	Revision int `json:"revision"`

	// Status of the release
	Status string `json:"status"`

	// FirstDeployed timestamp
	// +optional
	FirstDeployed *metav1.Time `json:"firstDeployed,omitempty"`

	// LastDeployed timestamp
	// +optional
	LastDeployed *metav1.Time `json:"lastDeployed,omitempty"`

	// Description of the release
	// +optional
	Description string `json:"description,omitempty"`

	// Chart name and version
	// +optional
	Chart string `json:"chart,omitempty"`

	// AppVersion of the application
	// +optional
	AppVersion string `json:"appVersion,omitempty"`
}

// FailureRecord contains information about a failed operation
type FailureRecord struct {
	// Time when the failure occurred
	Time metav1.Time `json:"time"`

	// Reason for the failure
	Reason string `json:"reason"`

	// Message describing the failure
	Message string `json:"message"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced
// +kubebuilder:printcolumn:name="Chart",type=string,JSONPath=`.spec.chart.name`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.chart.version`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// HelmRelease is the Schema for the helmreleases API.
type HelmRelease struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HelmReleaseSpec   `json:"spec,omitempty"`
	Status HelmReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HelmReleaseList contains a list of HelmRelease.
type HelmReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []HelmRelease `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HelmRelease{}, &HelmReleaseList{})
}
