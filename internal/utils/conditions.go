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

package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition types for HelmRepository
const (
	// RepositoryConditionReady indicates the repository is ready and synced
	RepositoryConditionReady = "Ready"
	// RepositoryConditionSyncing indicates the repository is being synced
	RepositoryConditionSyncing = "Syncing"
	// RepositoryConditionFailed indicates the repository sync failed
	RepositoryConditionFailed = "Failed"
)

// Condition types for HelmRelease
const (
	// ReleaseConditionReady indicates the release is ready
	ReleaseConditionReady = "Ready"
	// ReleaseConditionReleased indicates the release is deployed
	ReleaseConditionReleased = "Released"
	// ReleaseConditionFailed indicates the release operation failed
	ReleaseConditionFailed = "Failed"
	// ReleaseConditionProgressing indicates the release is being processed
	ReleaseConditionProgressing = "Progressing"
)

// Condition reasons
const (
	// Repository reasons
	ReasonRepositoryAdded      = "RepositoryAdded"
	ReasonRepositoryUpdated    = "RepositoryUpdated"
	ReasonSyncStarted          = "SyncStarted"
	ReasonSyncCompleted        = "SyncCompleted"
	ReasonSyncFailed           = "SyncFailed"
	ReasonRepositoryNotFound   = "RepositoryNotFound"
	ReasonInvalidURL           = "InvalidURL"
	ReasonAuthenticationFailed = "AuthenticationFailed"
	ReasonSuspended            = "Suspended"

	// Release reasons
	ReasonInstallStarted     = "InstallStarted"
	ReasonInstallCompleted   = "InstallCompleted"
	ReasonInstallFailed      = "InstallFailed"
	ReasonUpgradeStarted     = "UpgradeStarted"
	ReasonUpgradeCompleted   = "UpgradeCompleted"
	ReasonUpgradeFailed      = "UpgradeFailed"
	ReasonUninstallStarted   = "UninstallStarted"
	ReasonUninstallCompleted = "UninstallCompleted"
	ReasonUninstallFailed    = "UninstallFailed"
	ReasonChartNotFound      = "ChartNotFound"
	ReasonDependencyNotReady = "DependencyNotReady"
	ReasonConfigurationError = "ConfigurationError"
	ReasonReleaseSuspended   = "ReleaseSuspended"
)

// NewReadyCondition creates a new Ready condition
func NewReadyCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               RepositoryConditionReady,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// NewSyncingCondition creates a new Syncing condition
func NewSyncingCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               RepositoryConditionSyncing,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// NewFailedCondition creates a new Failed condition
func NewFailedCondition(reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               RepositoryConditionFailed,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// NewReleaseReadyCondition creates a new Ready condition for releases
func NewReleaseReadyCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               ReleaseConditionReady,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// NewReleaseReleasedCondition creates a new Released condition
func NewReleaseReleasedCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               ReleaseConditionReleased,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// NewReleaseProgressingCondition creates a new Progressing condition
func NewReleaseProgressingCondition(status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               ReleaseConditionProgressing,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

// NewReleaseFailedCondition creates a new Failed condition for releases
func NewReleaseFailedCondition(reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               ReleaseConditionFailed,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}
