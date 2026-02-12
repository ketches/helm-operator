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
	"time"

	"github.com/go-logr/logr"
	"helm.sh/helm/v3/pkg/repo"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	helmoperatorv1alpha1 "github.com/ketches/helm-operator/api/v1alpha1"
	"github.com/ketches/helm-operator/internal/helm"
	"github.com/ketches/helm-operator/internal/utils"
)

// HelmRepositoryReconciler reconciles a HelmRepository object
type HelmRepositoryReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Recorder   events.EventRecorder
	HelmClient helm.Client
}

// +kubebuilder:rbac:groups=helm-operator.ketches.cn,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm-operator.ketches.cn,resources=helmrepositories/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=helm-operator.ketches.cn,resources=helmrepositories/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop
func (r *HelmRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrepository", req.NamespacedName)

	// 1. Get HelmRepository resource
	repo := &helmoperatorv1alpha1.HelmRepository{}
	if err := r.Get(ctx, req.NamespacedName, repo); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("HelmRepository resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get HelmRepository")
		return ctrl.Result{}, err
	}

	// 2. Handle deletion logic
	if !repo.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, repo)
	}

	// 3. Add Finalizer
	if !controllerutil.ContainsFinalizer(repo, utils.HelmRepositoryFinalizer) {
		controllerutil.AddFinalizer(repo, utils.HelmRepositoryFinalizer)
		if err := r.Update(ctx, repo); err != nil {
			logger.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// 4. Execute main logic
	return r.reconcileNormal(ctx, repo)
}

// reconcileNormal handles the normal reconciliation logic
func (r *HelmRepositoryReconciler) reconcileNormal(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrepository", repo.Name, "namespace", repo.Namespace)

	// Validate configuration
	if err := r.validateSpec(repo); err != nil {
		logger.Error(err, "Invalid repository specification")
		condition := utils.NewFailedCondition(utils.ReasonConfigurationError, err.Error())
		if updateErr := r.updateStatusWithRetry(ctx, repo, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		r.Recorder.Eventf(repo, nil, "Warning", utils.ReasonConfigurationError, "configure", "%s", err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Check if suspended
	if repo.Spec.Suspend {
		logger.Info("Repository sync is suspended")
		condition := utils.NewReadyCondition(metav1.ConditionFalse, utils.ReasonSuspended, "Repository sync is suspended")
		if err := r.updateStatusWithRetry(ctx, repo, condition); err != nil {
			logger.Error(err, "Failed to update status")
		}
		return ctrl.Result{}, nil
	}

	// Check if sync is needed
	if !r.shouldSync(repo) {
		nextSync := r.calculateNextSync(repo)
		logger.V(1).Info("Repository sync not needed yet", "nextSync", nextSync)
		return ctrl.Result{RequeueAfter: nextSync}, nil
	}

	// Execute sync
	return r.reconcileSync(ctx, repo)
}

// reconcileSync performs the repository synchronization
func (r *HelmRepositoryReconciler) reconcileSync(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrepository", repo.Name, "namespace", repo.Namespace)

	logger.Info("Starting repository sync", "url", repo.Spec.URL, "type", repo.Spec.Type)

	// Set syncing status
	syncingCondition := utils.NewSyncingCondition(metav1.ConditionTrue, utils.ReasonSyncStarted, "Starting repository sync")
	if err := r.updateStatusWithRetry(ctx, repo, syncingCondition); err != nil {
		logger.Error(err, "Failed to update syncing status")
	}
	r.Recorder.Eventf(repo, nil, "Normal", utils.ReasonSyncStarted, "sync", "Starting repository sync")

	// Get authentication info
	auth, err := r.getRepositoryAuth(ctx, repo)
	if err != nil {
		logger.Error(err, "Failed to get repository authentication")
		condition := utils.NewFailedCondition(utils.ReasonAuthenticationFailed, err.Error())
		if updateErr := r.updateStatusWithRetry(ctx, repo, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		r.Recorder.Eventf(repo, nil, "Warning", utils.ReasonAuthenticationFailed, "authenticate", "%s", err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Add/update repository
	if err := r.addRepositoryToHelm(ctx, repo, auth); err != nil {
		logger.Error(err, "Failed to add repository")
		condition := utils.NewFailedCondition(utils.ReasonSyncFailed, fmt.Sprintf("Failed to add repository: %v", err))
		if updateErr := r.updateStatusWithRetry(ctx, repo, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		r.Recorder.Eventf(repo, nil, "Warning", utils.ReasonSyncFailed, "sync", "%s", err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Handle OCI repositories differently
	if r.isOCIRepository(repo) {
		logger.Info("OCI repository registered successfully")
		// For OCI repos, we can't fetch a chart list ahead of time
		// Just mark as synced successfully
		condition := utils.NewReadyCondition(metav1.ConditionTrue, utils.ReasonSyncCompleted, "OCI repository registered successfully")
		if err := r.updateStatusWithRetry(ctx, repo, condition); err != nil {
			logger.Error(err, "Failed to update status")
			return ctrl.Result{RequeueAfter: time.Minute}, err
		}
		r.Recorder.Eventf(repo, nil, "Normal", utils.ReasonSyncCompleted, "sync", "OCI repository registered successfully")

		// Calculate next sync time
		nextSync := r.calculateNextSync(repo)
		return ctrl.Result{RequeueAfter: nextSync}, nil
	}

	// Get charts information for traditional Helm repositories
	charts, err := r.HelmClient.GetChartsFromRepository(ctx, repo.Name)
	if err != nil {
		logger.Error(err, "Failed to get charts from repository")
		condition := utils.NewFailedCondition(utils.ReasonSyncFailed, fmt.Sprintf("Failed to get charts: %v", err))
		if updateErr := r.updateStatusWithRetry(ctx, repo, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		r.Recorder.Eventf(repo, nil, "Warning", utils.ReasonSyncFailed, "sync", "%s", err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Create ConfigMaps for chart values
	if err := r.createChartValuesConfigMaps(ctx, repo, charts); err != nil {
		logger.Error(err, "Failed to create chart values ConfigMaps")
		condition := utils.NewFailedCondition(utils.ReasonSyncFailed, fmt.Sprintf("Failed to create ConfigMaps: %v", err))
		if updateErr := r.updateStatusWithRetry(ctx, repo, condition); updateErr != nil {
			logger.Error(updateErr, "Failed to update status")
		}
		r.Recorder.Eventf(repo, nil, "Warning", utils.ReasonSyncFailed, "sync", "%s", err.Error())
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	// Update status with retry for conflicts
	if err := r.updateRepositoryStatusWithRetry(ctx, repo, charts); err != nil {
		logger.Error(err, "Failed to update repository status")
		return ctrl.Result{RequeueAfter: time.Minute}, err
	}

	logger.Info("Repository sync completed successfully", "chartsCount", len(charts))
	r.Recorder.Eventf(repo, nil, "Normal", utils.ReasonSyncCompleted, "sync", "Repository synced successfully, found %d charts", len(charts))

	// Calculate next sync time
	nextSync := r.calculateNextSync(repo)
	return ctrl.Result{RequeueAfter: nextSync}, nil
}

// isOCIRepository checks if the repository is an OCI registry
func (r *HelmRepositoryReconciler) isOCIRepository(repo *helmoperatorv1alpha1.HelmRepository) bool {
	return repo.Spec.Type == "oci" || (len(repo.Spec.URL) > 6 && repo.Spec.URL[:6] == "oci://")
}

// reconcileDelete handles the deletion logic
func (r *HelmRepositoryReconciler) reconcileDelete(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository) (ctrl.Result, error) {
	logger := r.Log.WithValues("helmrepository", repo.Name, "namespace", repo.Namespace)

	logger.Info("Deleting HelmRepository")

	// Remove repository from Helm
	if err := r.HelmClient.RemoveRepository(ctx, repo.Name); err != nil {
		logger.Error(err, "Failed to remove repository from Helm")
		// Don't block deletion, just log error
	}

	// Remove Finalizer with retry
	if err := r.removeFinalizerWithRetry(ctx, repo); err != nil {
		logger.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	logger.Info("HelmRepository deleted successfully")
	return ctrl.Result{}, nil
}

// validateSpec validates the repository specification
func (r *HelmRepositoryReconciler) validateSpec(repo *helmoperatorv1alpha1.HelmRepository) error {
	if repo.Spec.URL == "" {
		return fmt.Errorf("repository URL is required")
	}
	return nil
}

// shouldSync determines if the repository should be synced
func (r *HelmRepositoryReconciler) shouldSync(repo *helmoperatorv1alpha1.HelmRepository) bool {
	// Always sync if never synced before
	if repo.Status.LastSyncTime == nil {
		return true
	}

	// Check if the repository exists in local Helm configuration
	ctx := context.Background()
	localRepos, err := r.HelmClient.ListRepositories(ctx)
	if err != nil {
		// If we can't list repositories, assume we need to sync
		r.Log.Error(err, "Failed to list local repositories, assuming sync needed", "repository", repo.Name)
		return true
	}

	// Check if the repository exists locally
	repoExists := false
	for _, localRepo := range localRepos {
		if localRepo.Name == repo.Name {
			repoExists = true
			break
		}
	}

	// If repository doesn't exist locally, we need to sync
	if !repoExists {
		r.Log.Info("Repository not found locally, sync needed", "repository", repo.Name)
		return true
	}

	// Check time-based sync interval
	interval := r.getSyncInterval(repo)
	nextSync := repo.Status.LastSyncTime.Add(interval)
	return time.Now().After(nextSync)
}

// calculateNextSync calculates when the next sync should occur
func (r *HelmRepositoryReconciler) calculateNextSync(repo *helmoperatorv1alpha1.HelmRepository) time.Duration {
	if repo.Status.LastSyncTime == nil {
		return 0
	}

	interval := r.getSyncInterval(repo)
	nextSync := repo.Status.LastSyncTime.Add(interval)
	remaining := time.Until(nextSync)

	if remaining < 0 {
		return 0
	}
	return remaining
}

// getSyncInterval returns the sync interval for the repository
func (r *HelmRepositoryReconciler) getSyncInterval(repo *helmoperatorv1alpha1.HelmRepository) time.Duration {
	if repo.Spec.Interval == "" {
		return 30 * time.Minute // default interval
	}

	duration, err := time.ParseDuration(repo.Spec.Interval)
	if err != nil {
		return 30 * time.Minute // fallback to default
	}

	return duration
}

// getRepositoryAuth retrieves authentication information for the repository
func (r *HelmRepositoryReconciler) getRepositoryAuth(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository) (*RepositoryAuth, error) {
	auth := &RepositoryAuth{}

	if repo.Spec.Auth == nil {
		return auth, nil
	}

	// Handle basic authentication
	if repo.Spec.Auth.Basic != nil {
		if repo.Spec.Auth.Basic.Username != "" {
			auth.Username = repo.Spec.Auth.Basic.Username
		}
		if repo.Spec.Auth.Basic.Password != "" {
			auth.Password = repo.Spec.Auth.Basic.Password
		}

		// Handle secret reference
		if repo.Spec.Auth.Basic.SecretRef != nil {
			secretAuth, err := r.getAuthFromSecret(ctx, repo.Spec.Auth.Basic.SecretRef)
			if err != nil {
				return nil, fmt.Errorf("failed to get auth from secret: %w", err)
			}
			if secretAuth.Username != "" {
				auth.Username = secretAuth.Username
			}
			if secretAuth.Password != "" {
				auth.Password = secretAuth.Password
			}
		}
	}

	// Handle TLS configuration
	if repo.Spec.Auth.TLS != nil {
		auth.InsecureSkipTLSverify = repo.Spec.Auth.TLS.InsecureSkipVerify
		auth.CAFile = repo.Spec.Auth.TLS.CAFile
		auth.CertFile = repo.Spec.Auth.TLS.CertFile
		auth.KeyFile = repo.Spec.Auth.TLS.KeyFile
	}

	return auth, nil
}

// getAuthFromSecret retrieves authentication credentials from a secret
func (r *HelmRepositoryReconciler) getAuthFromSecret(ctx context.Context, secretRef *helmoperatorv1alpha1.SecretReference) (*RepositoryAuth, error) {
	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: secretRef.Namespace,
	}

	if err := r.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	auth := &RepositoryAuth{}
	if username, ok := secret.Data["username"]; ok {
		auth.Username = string(username)
	}
	if password, ok := secret.Data["password"]; ok {
		auth.Password = string(password)
	}

	return auth, nil
}

// updateStatus updates the repository status
func (r *HelmRepositoryReconciler) updateStatus(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository, condition metav1.Condition) error {
	repo = repo.DeepCopy()

	meta.SetStatusCondition(&repo.Status.Conditions, condition)
	repo.Status.ObservedGeneration = repo.Generation

	return r.Status().Update(ctx, repo)
}

// updateRepositoryStatus updates the repository status with charts information
func (r *HelmRepositoryReconciler) updateRepositoryStatus(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository, charts []helm.ChartInfo) error {
	repo = repo.DeepCopy()

	// Convert charts to API format
	var chartInfos []helmoperatorv1alpha1.ChartInfo
	totalVersions := 0

	for _, chart := range charts {
		chartInfo := helmoperatorv1alpha1.ChartInfo{
			Name:        chart.Name,
			Description: chart.Description,
			Versions: []helmoperatorv1alpha1.ChartVersion{
				{
					Version:    chart.Version,
					AppVersion: chart.AppVersion,
					Created:    &metav1.Time{Time: chart.Created},
					Digest:     chart.Digest,
				},
			},
		}
		chartInfos = append(chartInfos, chartInfo)
		totalVersions++
	}

	// Update status
	repo.Status.Charts = chartInfos
	repo.Status.Stats = &helmoperatorv1alpha1.RepositoryStats{
		TotalCharts:   len(chartInfos),
		TotalVersions: totalVersions,
	}
	repo.Status.LastSyncTime = &metav1.Time{Time: time.Now()}
	repo.Status.ObservedGeneration = repo.Generation

	// Set ready condition
	condition := utils.NewReadyCondition(metav1.ConditionTrue, utils.ReasonSyncCompleted, fmt.Sprintf("Repository synced successfully, found %d charts", len(chartInfos)))
	meta.SetStatusCondition(&repo.Status.Conditions, condition)

	return r.Status().Update(ctx, repo)
}

// updateRepositoryStatusWithRetry updates repository status with retry for conflicts
func (r *HelmRepositoryReconciler) updateRepositoryStatusWithRetry(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository, charts []helm.ChartInfo) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get the latest version of the resource
		latest := &helmoperatorv1alpha1.HelmRepository{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(repo), latest); err != nil {
			return err
		}

		return r.updateRepositoryStatus(ctx, latest, charts)
	})
}

// updateStatusWithRetry updates status with retry for conflicts
func (r *HelmRepositoryReconciler) updateStatusWithRetry(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository, condition metav1.Condition) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get the latest version of the resource
		latest := &helmoperatorv1alpha1.HelmRepository{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(repo), latest); err != nil {
			return err
		}

		return r.updateStatus(ctx, latest, condition)
	})
}

// RepositoryAuth contains authentication information
type RepositoryAuth struct {
	Username              string
	Password              string
	CAFile                string
	CertFile              string
	KeyFile               string
	InsecureSkipTLSverify bool
}

// addRepositoryToHelm adds a repository to Helm with proper authentication
func (r *HelmRepositoryReconciler) addRepositoryToHelm(ctx context.Context, repository *helmoperatorv1alpha1.HelmRepository, auth *RepositoryAuth) error {
	// Create repository entry
	entry := &repo.Entry{
		Name:                  repository.Name,
		URL:                   repository.Spec.URL,
		Username:              auth.Username,
		Password:              auth.Password,
		CertFile:              auth.CertFile,
		KeyFile:               auth.KeyFile,
		CAFile:                auth.CAFile,
		InsecureSkipTLSverify: auth.InsecureSkipTLSverify,
	}

	return r.HelmClient.AddRepository(ctx, entry)
}

// createChartValuesConfigMaps creates ConfigMaps for all chart versions' values.yaml
func (r *HelmRepositoryReconciler) createChartValuesConfigMaps(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository, charts []helm.ChartInfo) error {
	logger := r.Log.WithValues("helmrepository", repo.Name, "namespace", repo.Namespace)

	// Check ConfigMap policy
	policy := repo.Spec.ValuesConfigMapPolicy
	if policy == "disabled" || policy == "" {
		logger.V(1).Info("ConfigMap generation is disabled")
		return nil
	}

	// For on-demand policy, skip generation during sync
	// ConfigMaps will be created when HelmRelease references them
	if policy == "on-demand" {
		logger.V(1).Info("Using on-demand ConfigMap generation policy, skipping sync-time generation")
		return nil
	}

	// For lazy policy, only generate for latest versions
	logger.V(1).Info("Creating ConfigMaps for chart values", "policy", policy)

	for _, chart := range charts {
		// For lazy policy, only create ConfigMap for latest version
		if policy == "lazy" {
			if err := r.createChartVersionConfigMap(ctx, repo, chart.Name, chart.Version); err != nil {
				logger.Error(err, "Failed to create ConfigMap for latest chart version",
					"chartName", chart.Name, "version", chart.Version)
			}
			continue
		}

		// Full generation for other policies (shouldn't reach here with current policies)
		chartVersions, err := r.HelmClient.GetChartVersions(ctx, repo.Name, chart.Name)
		if err != nil {
			logger.Error(err, "Failed to get chart versions", "chartName", chart.Name)
			continue
		}

		for _, version := range chartVersions {
			if err := r.createChartVersionConfigMap(ctx, repo, chart.Name, version.Version); err != nil {
				logger.Error(err, "Failed to create ConfigMap for chart version",
					"chartName", chart.Name, "version", version.Version)
			}
		}
	}

	// Cleanup old ConfigMaps based on retention policy
	if err := r.cleanupOldConfigMaps(ctx, repo); err != nil {
		logger.Error(err, "Failed to cleanup old ConfigMaps")
	}

	return nil
}

// createChartVersionConfigMap creates a ConfigMap for a specific chart version's values.yaml
func (r *HelmRepositoryReconciler) createChartVersionConfigMap(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository, chartName, version string) error {
	logger := r.Log.WithValues("helmrepository", repo.Name, "chartName", chartName, "version", version)

	// Get chart values
	values, err := r.HelmClient.GetChartValues(ctx, repo.Name, chartName, version)
	if err != nil {
		return fmt.Errorf("failed to get chart values: %w", err)
	}

	// Generate ConfigMap name
	configMapName := r.generateConfigMapName(repo.Name, chartName, version)

	// Check if ConfigMap already exists
	existingConfigMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      configMapName,
		Namespace: repo.Namespace,
	}, existingConfigMap)

	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to check existing ConfigMap: %w", err)
	}

	// Create ConfigMap object
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: repo.Namespace,
			Labels: map[string]string{
				"ketches.cn/owned":                    "true",
				"helm-operator.ketches.cn/repository": repo.Name,
				"helm-operator.ketches.cn/chart":      chartName,
				"helm-operator.ketches.cn/version":    version,
			},
		},
		Data: map[string]string{
			"values.yaml": values,
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(repo, configMap, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if apierrors.IsNotFound(err) {
		// ConfigMap doesn't exist, create it
		logger.Info("Creating ConfigMap for chart values")
		if err := r.Create(ctx, configMap); err != nil {
			return fmt.Errorf("failed to create ConfigMap: %w", err)
		}
	} else {
		// ConfigMap exists, update it if values changed
		if existingConfigMap.Data["values.yaml"] != values {
			logger.Info("Updating ConfigMap for chart values")
			existingConfigMap.Data = configMap.Data
			existingConfigMap.Labels = configMap.Labels

			if err := r.Update(ctx, existingConfigMap); err != nil {
				return fmt.Errorf("failed to update ConfigMap: %w", err)
			}
		}
	}

	return nil
}

// generateConfigMapName generates a consistent name for chart values ConfigMap
func (r *HelmRepositoryReconciler) generateConfigMapName(repoName, chartName, version string) string {
	// Format: helm-values-{repo}-{chart}-{version}
	// Replace dots and other special characters with dashes for valid Kubernetes names
	safeName := fmt.Sprintf("helm-values-%s-%s-%s", repoName, chartName, version)

	// Replace invalid characters
	safeName = sanitizeKubernetesName(safeName)

	// Ensure name is not too long (max 253 characters for ConfigMap names)
	if len(safeName) > 253 {
		// Truncate and add hash to ensure uniqueness
		hash := fmt.Sprintf("%x", []byte(safeName))[:8]
		safeName = safeName[:240] + "-" + hash
	}

	return safeName
}

// cleanupOldConfigMaps removes ConfigMaps that have exceeded the retention period.
func (r *HelmRepositoryReconciler) cleanupOldConfigMaps(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository) error {
	retention := repo.Spec.ValuesConfigMapRetention
	if retention == "" {
		retention = "168h"
	}
	retentionDuration, err := time.ParseDuration(retention)
	if err != nil {
		return fmt.Errorf("invalid retention duration: %w", err)
	}
	configMapList := &corev1.ConfigMapList{}
	if err := r.List(ctx, configMapList, client.InNamespace(repo.Namespace),
		client.MatchingLabels{"helm-operator.ketches.cn/repository": repo.Name}); err != nil {
		return fmt.Errorf("failed to list ConfigMaps: %w", err)
	}
	now := time.Now()
	for i := range configMapList.Items {
		cm := &configMapList.Items[i]
		if now.Sub(cm.CreationTimestamp.Time) > retentionDuration {
			if err := r.Delete(ctx, cm); err != nil {
				r.Log.Error(err, "Failed to delete old ConfigMap", "configMap", cm.Name)
			}
		}
	}
	return nil
}

// sanitizeKubernetesName replaces invalid characters in Kubernetes resource names
func sanitizeKubernetesName(name string) string {
	// Replace dots, plus signs, and other invalid characters with dashes
	result := ""
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			result += string(char)
		} else {
			result += "-"
		}
	}

	// Remove leading/trailing dashes and consecutive dashes
	for len(result) > 0 && result[0] == '-' {
		result = result[1:]
	}
	for len(result) > 0 && result[len(result)-1] == '-' {
		result = result[:len(result)-1]
	}

	// Replace consecutive dashes with single dash
	for i := 0; i < len(result)-1; i++ {
		if result[i] == '-' && result[i+1] == '-' {
			result = result[:i] + result[i+1:]
			i-- // Check the same position again
		}
	}

	return result
}

// removeFinalizerWithRetry removes finalizer with retry mechanism to handle conflicts
func (r *HelmRepositoryReconciler) removeFinalizerWithRetry(ctx context.Context, repo *helmoperatorv1alpha1.HelmRepository) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// Get the latest version of the resource
		latest := &helmoperatorv1alpha1.HelmRepository{}
		if err := r.Get(ctx, client.ObjectKeyFromObject(repo), latest); err != nil {
			if apierrors.IsNotFound(err) {
				// Resource already deleted, consider success
				return nil
			}
			return err
		}

		// Remove finalizer from the latest version
		controllerutil.RemoveFinalizer(latest, utils.HelmRepositoryFinalizer)

		// Update the resource
		return r.Update(ctx, latest)
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&helmoperatorv1alpha1.HelmRepository{}).
		Owns(&corev1.ConfigMap{}).
		Named("helmrepository").
		Complete(r)
}
