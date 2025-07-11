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

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmoperatorv1alpha1 "github.com/ketches/helm-operator/api/v1alpha1"
	"github.com/ketches/helm-operator/internal/helm"
	"github.com/ketches/helm-operator/internal/utils"
)

// HelmReleaseReconciler reconciles a HelmRelease object
type HelmReleaseReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	HelmClient helm.Client
}

// +kubebuilder:rbac:groups=helm-operator.ketches.cn,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm-operator.ketches.cn,resources=helmreleases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm-operator.ketches.cn,resources=helmreleases/finalizers,verbs=update
// +kubebuilder:rbac:groups=helm-operator.ketches.cn,resources=helmrepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create

// Reconcile is part of the main kubernetes reconciliation loop
func (r *HelmReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrelease", req.NamespacedName)

	// 1. Get HelmRelease resource
	release := &helmoperatorv1alpha1.HelmRelease{}
	if err := r.Get(ctx, req.NamespacedName, release); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("HelmRelease resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get HelmRelease")
		return ctrl.Result{}, err
	}

	// 2. Handle deletion logic
	if !release.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, release)
	}

	// 3. Add Finalizer
	if !controllerutil.ContainsFinalizer(release, utils.HelmReleaseFinalizer) {
		controllerutil.AddFinalizer(release, utils.HelmReleaseFinalizer)
		if err := r.Update(ctx, release); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 4. Execute main logic
	return r.reconcileNormal(ctx, release)
}

// reconcileNormal handles the normal reconciliation logic
func (r *HelmReleaseReconciler) reconcileNormal(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrelease", release.Name, "namespace", release.Namespace)

	// Validate configuration
	if err := r.validateSpec(release); err != nil {
		logger.Error(err, "Invalid release specification")
		condition := utils.NewReleaseFailedCondition(utils.ReasonConfigurationError, err.Error())
		if updateErr := r.updateStatus(ctx, release, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		r.Recorder.Event(release, "Warning", utils.ReasonConfigurationError, err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Check if suspended
	if release.Spec.Suspend {
		logger.Info("Release is suspended")
		condition := utils.NewReleaseReadyCondition(metav1.ConditionFalse, utils.ReasonReleaseSuspended, "Release is suspended")
		if err := r.updateStatus(ctx, release, condition); err != nil {
			logger.Error(err, "Failed to update status")
		}
		return ctrl.Result{}, nil
	}

	// Check dependencies (HelmRepository)
	if err := r.checkDependencies(ctx, release); err != nil {
		logger.Error(err, "Dependencies not ready")
		condition := utils.NewReleaseFailedCondition(utils.ReasonDependencyNotReady, err.Error())
		if updateErr := r.updateStatus(ctx, release, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}

	// Execute release reconciliation
	return r.reconcileRelease(ctx, release)
}

// reconcileRelease handles the actual release operations
func (r *HelmReleaseReconciler) reconcileRelease(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrelease", release.Name, "namespace", release.Namespace)

	// Get release name and namespace
	releaseName := r.getReleaseName(release)
	releaseNamespace := r.getReleaseNamespace(release)

	logger.Info("Reconciling Helm release", "releaseName", releaseName, "releaseNamespace", releaseNamespace)

	// Check if release exists
	existingRelease, err := r.HelmClient.GetRelease(ctx, releaseName, releaseNamespace)
	if err != nil && !isReleaseNotFoundError(err) {
		logger.Error(err, "Failed to get existing release")
		condition := utils.NewReleaseFailedCondition(utils.ReasonInstallFailed, fmt.Sprintf("Failed to get release: %v", err))
		if updateErr := r.updateStatus(ctx, release, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	if existingRelease == nil {
		// Release doesn't exist, install it
		return r.installRelease(ctx, release)
	} else {
		// Release exists, check if upgrade is needed
		return r.upgradeReleaseIfNeeded(ctx, release, existingRelease)
	}
}

// reconcileDelete handles the deletion logic
func (r *HelmReleaseReconciler) reconcileDelete(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrelease", release.Name, "namespace", release.Namespace)

	logger.Info("Deleting HelmRelease")

	// Get release name and namespace
	releaseName := r.getReleaseName(release)
	releaseNamespace := r.getReleaseNamespace(release)

	// Uninstall Helm release
	uninstallReq := &helm.UninstallRequest{
		Name:         releaseName,
		Namespace:    releaseNamespace,
		Timeout:      r.getUninstallTimeout(release),
		DisableHooks: r.getUninstallDisableHooks(release),
		KeepHistory:  r.getUninstallKeepHistory(release),
	}

	if err := r.HelmClient.UninstallRelease(ctx, uninstallReq); err != nil {
		if !isReleaseNotFoundError(err) {
			logger.Error(err, "Failed to uninstall release")
			// Don't block deletion, just log error
		}
	}

	// Remove Finalizer
	controllerutil.RemoveFinalizer(release, utils.HelmReleaseFinalizer)
	if err := r.Update(ctx, release); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("HelmRelease deleted successfully")
	return ctrl.Result{}, nil
}

// validateSpec validates the release specification
func (r *HelmReleaseReconciler) validateSpec(release *helmoperatorv1alpha1.HelmRelease) error {
	if release.Spec.Chart.Name == "" {
		return fmt.Errorf("chart name is required")
	}

	if release.Spec.Chart.Repository == nil && release.Spec.Chart.RepositoryURL == "" {
		return fmt.Errorf("chart repository or repositoryURL is required")
	}

	return nil
}

// checkDependencies checks if the HelmRepository dependency is ready
func (r *HelmReleaseReconciler) checkDependencies(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease) error {
	if release.Spec.Chart.Repository == nil {
		// Using direct repository URL, no dependency check needed
		return nil
	}

	// Get the HelmRepository
	repoNamespace := release.Spec.Chart.Repository.Namespace
	if repoNamespace == "" {
		repoNamespace = release.Namespace
	}

	repo := &helmoperatorv1alpha1.HelmRepository{}
	repoKey := types.NamespacedName{
		Name:      release.Spec.Chart.Repository.Name,
		Namespace: repoNamespace,
	}

	if err := r.Get(ctx, repoKey, repo); err != nil {
		return fmt.Errorf("failed to get HelmRepository %s: %w", repoKey, err)
	}

	// Check if repository is ready
	if !meta.IsStatusConditionTrue(repo.Status.Conditions, utils.RepositoryConditionReady) {
		return fmt.Errorf("HelmRepository %s is not ready", repoKey)
	}

	return nil
}

// installRelease installs a new Helm release
func (r *HelmReleaseReconciler) installRelease(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrelease", release.Name, "namespace", release.Namespace)

	logger.Info("Installing Helm release")

	// Set progressing condition
	condition := utils.NewReleaseProgressingCondition(metav1.ConditionTrue, utils.ReasonInstallStarted, "Starting release installation")
	if err := r.updateStatus(ctx, release, condition); err != nil {
		logger.Error(err, "Failed to update status")
	}
	r.Recorder.Event(release, "Normal", utils.ReasonInstallStarted, "Starting release installation")

	// Prepare install request
	installReq := &helm.InstallRequest{
		Name:            r.getReleaseName(release),
		Namespace:       r.getReleaseNamespace(release),
		Chart:           r.getChartReference(release),
		Version:         release.Spec.Chart.Version,
		Values:          release.Spec.Values,
		CreateNamespace: r.getCreateNamespace(release),
		Wait:            r.getInstallWait(release),
		WaitForJobs:     r.getInstallWaitForJobs(release),
		Timeout:         r.getInstallTimeout(release),
		SkipCRDs:        r.getInstallSkipCRDs(release),
		Replace:         r.getInstallReplace(release),
		DisableHooks:    r.getInstallDisableHooks(release),
	}

	// Install release
	releaseInfo, err := r.HelmClient.InstallRelease(ctx, installReq)
	if err != nil {
		logger.Error(err, "Failed to install release")
		condition := utils.NewReleaseFailedCondition(utils.ReasonInstallFailed, fmt.Sprintf("Failed to install: %v", err))
		if updateErr := r.updateStatus(ctx, release, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		r.Recorder.Event(release, "Warning", utils.ReasonInstallFailed, err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Update status with successful installation
	if err := r.updateReleaseStatus(ctx, release, releaseInfo); err != nil {
		logger.Error(err, "Failed to update release status")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	logger.Info("Release installed successfully")
	r.Recorder.Event(release, "Normal", utils.ReasonInstallCompleted, "Release installed successfully")

	// Calculate next reconciliation time if interval is set
	nextReconcile := r.calculateNextReconcile(release)
	return ctrl.Result{RequeueAfter: nextReconcile}, nil
}

// upgradeReleaseIfNeeded checks if upgrade is needed and performs it
func (r *HelmReleaseReconciler) upgradeReleaseIfNeeded(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease, existingRelease *helm.ReleaseInfo) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrelease", release.Name, "namespace", release.Namespace)

	// Check if upgrade is needed
	needsUpgrade, reason := r.needsUpgrade(release, existingRelease)
	if !needsUpgrade {
		logger.V(1).Info("No upgrade needed")

		// Update status to reflect current state
		if err := r.updateReleaseStatus(ctx, release, existingRelease); err != nil {
			logger.Error(err, "Failed to update release status")
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}

		// Calculate next reconciliation time
		nextReconcile := r.calculateNextReconcile(release)
		return ctrl.Result{RequeueAfter: nextReconcile}, nil
	}

	logger.Info("Upgrading Helm release", "reason", reason)

	// Set progressing condition
	condition := utils.NewReleaseProgressingCondition(metav1.ConditionTrue, utils.ReasonUpgradeStarted, fmt.Sprintf("Starting release upgrade: %s", reason))
	if err := r.updateStatus(ctx, release, condition); err != nil {
		logger.Error(err, "Failed to update status")
	}
	r.Recorder.Event(release, "Normal", utils.ReasonUpgradeStarted, fmt.Sprintf("Starting release upgrade: %s", reason))

	// Prepare upgrade request
	upgradeReq := &helm.UpgradeRequest{
		Name:          r.getReleaseName(release),
		Namespace:     r.getReleaseNamespace(release),
		Chart:         r.getChartReference(release),
		Version:       release.Spec.Chart.Version,
		Values:        release.Spec.Values,
		Wait:          r.getUpgradeWait(release),
		WaitForJobs:   r.getUpgradeWaitForJobs(release),
		Timeout:       r.getUpgradeTimeout(release),
		Force:         r.getUpgradeForce(release),
		ResetValues:   r.getUpgradeResetValues(release),
		ReuseValues:   r.getUpgradeReuseValues(release),
		Recreate:      r.getUpgradeRecreate(release),
		MaxHistory:    r.getUpgradeMaxHistory(release),
		CleanupOnFail: r.getUpgradeCleanupOnFail(release),
		DisableHooks:  r.getUpgradeDisableHooks(release),
	}

	// Upgrade release
	releaseInfo, err := r.HelmClient.UpgradeRelease(ctx, upgradeReq)
	if err != nil {
		logger.Error(err, "Failed to upgrade release")
		condition := utils.NewReleaseFailedCondition(utils.ReasonUpgradeFailed, fmt.Sprintf("Failed to upgrade: %v", err))
		if updateErr := r.updateStatus(ctx, release, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		r.Recorder.Event(release, "Warning", utils.ReasonUpgradeFailed, err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Update status with successful upgrade
	if err := r.updateReleaseStatus(ctx, release, releaseInfo); err != nil {
		logger.Error(err, "Failed to update release status")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	logger.Info("Release upgraded successfully")
	r.Recorder.Event(release, "Normal", utils.ReasonUpgradeCompleted, "Release upgraded successfully")

	// Calculate next reconciliation time
	nextReconcile := r.calculateNextReconcile(release)
	return ctrl.Result{RequeueAfter: nextReconcile}, nil
}

// Helper methods for getting release configuration
func (r *HelmReleaseReconciler) getReleaseName(release *helmoperatorv1alpha1.HelmRelease) string {
	if release.Spec.Release != nil && release.Spec.Release.Name != "" {
		return release.Spec.Release.Name
	}
	return release.Name
}

func (r *HelmReleaseReconciler) getReleaseNamespace(release *helmoperatorv1alpha1.HelmRelease) string {
	if release.Spec.Release != nil && release.Spec.Release.Namespace != "" {
		return release.Spec.Release.Namespace
	}
	return release.Namespace
}

func (r *HelmReleaseReconciler) getCreateNamespace(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Release != nil {
		return release.Spec.Release.CreateNamespace
	}
	return false
}

func (r *HelmReleaseReconciler) getChartReference(release *helmoperatorv1alpha1.HelmRelease) string {
	if release.Spec.Chart.RepositoryURL != "" {
		// For direct repository URL, return the chart name and let Helm handle the URL
		return release.Spec.Chart.Name
	}

	// For repository reference, use the format repo_name/chart_name
	if release.Spec.Chart.Repository != nil {
		return fmt.Sprintf("%s/%s", release.Spec.Chart.Repository.Name, release.Spec.Chart.Name)
	}

	// Fallback to just chart name (for local charts or other cases)
	return release.Spec.Chart.Name
}

// Install configuration helpers
func (r *HelmReleaseReconciler) getInstallTimeout(release *helmoperatorv1alpha1.HelmRelease) time.Duration {
	if release.Spec.Install != nil && release.Spec.Install.Timeout != "" {
		if duration, err := time.ParseDuration(release.Spec.Install.Timeout); err == nil {
			return duration
		}
	}
	return 10 * time.Minute // default
}

func (r *HelmReleaseReconciler) getInstallWait(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Install != nil {
		return release.Spec.Install.Wait
	}
	return true // default
}

func (r *HelmReleaseReconciler) getInstallWaitForJobs(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Install != nil {
		return release.Spec.Install.WaitForJobs
	}
	return true // default
}

func (r *HelmReleaseReconciler) getInstallSkipCRDs(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Install != nil {
		return release.Spec.Install.SkipCRDs
	}
	return false // default
}

func (r *HelmReleaseReconciler) getInstallReplace(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Install != nil {
		return release.Spec.Install.Replace
	}
	return false // default
}

func (r *HelmReleaseReconciler) getInstallDisableHooks(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Install != nil {
		return release.Spec.Install.DisableHooks
	}
	return false // default
}

// Upgrade configuration helpers
func (r *HelmReleaseReconciler) getUpgradeTimeout(release *helmoperatorv1alpha1.HelmRelease) time.Duration {
	if release.Spec.Upgrade != nil && release.Spec.Upgrade.Timeout != "" {
		if duration, err := time.ParseDuration(release.Spec.Upgrade.Timeout); err == nil {
			return duration
		}
	}
	return 10 * time.Minute // default
}

func (r *HelmReleaseReconciler) getUpgradeWait(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Upgrade != nil {
		return release.Spec.Upgrade.Wait
	}
	return true // default
}

func (r *HelmReleaseReconciler) getUpgradeWaitForJobs(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Upgrade != nil {
		return release.Spec.Upgrade.WaitForJobs
	}
	return true // default
}

func (r *HelmReleaseReconciler) getUpgradeForce(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Upgrade != nil {
		return release.Spec.Upgrade.Force
	}
	return false // default
}

func (r *HelmReleaseReconciler) getUpgradeResetValues(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Upgrade != nil {
		return release.Spec.Upgrade.ResetValues
	}
	return false // default
}

func (r *HelmReleaseReconciler) getUpgradeReuseValues(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Upgrade != nil {
		return release.Spec.Upgrade.ReuseValues
	}
	return false // default
}

func (r *HelmReleaseReconciler) getUpgradeRecreate(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Upgrade != nil {
		return release.Spec.Upgrade.Recreate
	}
	return false // default
}

func (r *HelmReleaseReconciler) getUpgradeMaxHistory(release *helmoperatorv1alpha1.HelmRelease) int {
	if release.Spec.Upgrade != nil && release.Spec.Upgrade.MaxHistory > 0 {
		return release.Spec.Upgrade.MaxHistory
	}
	return 10 // default
}

func (r *HelmReleaseReconciler) getUpgradeCleanupOnFail(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Upgrade != nil {
		return release.Spec.Upgrade.CleanupOnFail
	}
	return true // default
}

func (r *HelmReleaseReconciler) getUpgradeDisableHooks(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Upgrade != nil {
		return release.Spec.Upgrade.DisableHooks
	}
	return false // default
}

// Uninstall configuration helpers
func (r *HelmReleaseReconciler) getUninstallTimeout(release *helmoperatorv1alpha1.HelmRelease) time.Duration {
	if release.Spec.Uninstall != nil && release.Spec.Uninstall.Timeout != "" {
		if duration, err := time.ParseDuration(release.Spec.Uninstall.Timeout); err == nil {
			return duration
		}
	}
	return 5 * time.Minute // default
}

func (r *HelmReleaseReconciler) getUninstallDisableHooks(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Uninstall != nil {
		return release.Spec.Uninstall.DisableHooks
	}
	return false // default
}

func (r *HelmReleaseReconciler) getUninstallKeepHistory(release *helmoperatorv1alpha1.HelmRelease) bool {
	if release.Spec.Uninstall != nil {
		return release.Spec.Uninstall.KeepHistory
	}
	return false // default
}

// Status and utility methods
func (r *HelmReleaseReconciler) updateStatusWithRetry(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease, updateFunc func(*helmoperatorv1alpha1.HelmRelease)) error {
	const maxRetries = 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		// Get the latest version of the resource
		latest := &helmoperatorv1alpha1.HelmRelease{}
		if err := r.Get(ctx, types.NamespacedName{Name: release.Name, Namespace: release.Namespace}, latest); err != nil {
			return err
		}

		// Apply the update function
		updateFunc(latest)

		// Try to update the status
		if err := r.Status().Update(ctx, latest); err != nil {
			lastErr = err
			// If it's a conflict error, retry
			if apierrors.IsConflict(err) {
				r.Log.V(1).Info("Resource version conflict, retrying", "attempt", i+1, "error", err)
				continue
			}
			// If it's not a conflict error, return immediately
			return err
		}

		// Success
		return nil
	}

	// All retries failed
	return fmt.Errorf("failed to update status after %d retries, last error: %w", maxRetries, lastErr)
}

func (r *HelmReleaseReconciler) updateStatus(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease, condition metav1.Condition) error {
	return r.updateStatusWithRetry(ctx, release, func(r *helmoperatorv1alpha1.HelmRelease) {
		meta.SetStatusCondition(&r.Status.Conditions, condition)
		r.Status.ObservedGeneration = r.Generation
	})
}

func (r *HelmReleaseReconciler) updateReleaseStatus(ctx context.Context, release *helmoperatorv1alpha1.HelmRelease, releaseInfo *helm.ReleaseInfo) error {
	return r.updateStatusWithRetry(ctx, release, func(r *helmoperatorv1alpha1.HelmRelease) {
		// Update Helm release information
		r.Status.HelmRelease = &helmoperatorv1alpha1.HelmReleaseInfo{
			Name:        releaseInfo.Name,
			Namespace:   releaseInfo.Namespace,
			Revision:    releaseInfo.Revision,
			Status:      releaseInfo.Status,
			Chart:       releaseInfo.Chart,
			AppVersion:  releaseInfo.AppVersion,
			Description: releaseInfo.Description,
		}

		if releaseInfo.FirstDeployed != nil {
			r.Status.HelmRelease.FirstDeployed = &metav1.Time{Time: *releaseInfo.FirstDeployed}
		}
		if releaseInfo.LastDeployed != nil {
			r.Status.HelmRelease.LastDeployed = &metav1.Time{Time: *releaseInfo.LastDeployed}
		}

		// Update last applied configuration
		r.Status.LastAppliedConfiguration = &release.Spec

		// Set ready condition
		condition := utils.NewReleaseReadyCondition(metav1.ConditionTrue, utils.ReasonInstallCompleted, "Release is ready")
		meta.SetStatusCondition(&r.Status.Conditions, condition)

		// Set released condition
		releasedCondition := utils.NewReleaseReleasedCondition(metav1.ConditionTrue, utils.ReasonInstallCompleted, "Release is deployed")
		meta.SetStatusCondition(&r.Status.Conditions, releasedCondition)

		r.Status.ObservedGeneration = r.Generation
	})
}

func (r *HelmReleaseReconciler) needsUpgrade(release *helmoperatorv1alpha1.HelmRelease, existingRelease *helm.ReleaseInfo) (bool, string) {
	// Check if chart version changed
	if release.Spec.Chart.Version != "" && !r.isVersionMatch(existingRelease.Chart, release.Spec.Chart.Version) {
		return true, fmt.Sprintf("chart version changed to %s", release.Spec.Chart.Version)
	}

	// Check if values changed
	if !r.areValuesEqual(release.Spec.Values, existingRelease.Values) {
		return true, "values configuration changed"
	}

	// Check if release configuration changed
	if release.Status.LastAppliedConfiguration != nil {
		if !r.isSpecEqual(&release.Spec, release.Status.LastAppliedConfiguration) {
			return true, "release configuration changed"
		}
	}

	return false, ""
}

func (r *HelmReleaseReconciler) isVersionMatch(chartInfo, requestedVersion string) bool {
	// Extract version from chart info (format: "chartname-version")
	parts := strings.Split(chartInfo, "-")
	if len(parts) < 2 {
		return false
	}
	currentVersion := parts[len(parts)-1]
	return currentVersion == requestedVersion
}

func (r *HelmReleaseReconciler) areValuesEqual(newValues string, existingValues string) bool {
	if newValues == "" && existingValues == "" {
		return true // Both are empty, considered equal
	}
	if newValues == "" || existingValues == "" {
		return false // One is empty, the other is not
	}

	var (
		newValuesM, existingValuesM map[string]any
	)
	if err := yaml.Unmarshal([]byte(newValues), &newValuesM); err != nil {
		return false
	}
	if err := yaml.Unmarshal([]byte(existingValues), &existingValuesM); err != nil {
		return false
	}
	return utils.MapEquals(newValuesM, existingValuesM)
	// Simple string comparison for now
	// In a more sophisticated implementation, we could parse YAML and do semantic comparison
	// return newValues == existingValues
}

func (r *HelmReleaseReconciler) isSpecEqual(spec1, spec2 *helmoperatorv1alpha1.HelmReleaseSpec) bool {
	// Simple comparison - in production, you might want more sophisticated comparison
	return spec1.Chart.Name == spec2.Chart.Name &&
		spec1.Chart.Version == spec2.Chart.Version &&
		r.areValuesEqual(spec1.Values, spec2.Values)
}

func (r *HelmReleaseReconciler) calculateNextReconcile(release *helmoperatorv1alpha1.HelmRelease) time.Duration {
	if release.Spec.Interval == "" {
		return 0 // No automatic reconciliation
	}

	duration, err := time.ParseDuration(release.Spec.Interval)
	if err != nil {
		return 0 // Invalid interval, no automatic reconciliation
	}

	return duration
}

func isReleaseNotFoundError(err error) bool {
	// Check if the error indicates that the release was not found
	// This is a simplified check - in practice, you'd check for specific Helm error types
	return err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "release: not found"))
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmoperatorv1alpha1.HelmRelease{}).
		Named("helmrelease").
		Complete(r)
}
