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

package helm

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
)

// InstallRelease installs a new Helm release
func (c *helmClient) InstallRelease(ctx context.Context, req *InstallRequest) (*ReleaseInfo, error) {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(req.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create action config for namespace %s: %w", req.Namespace, err)
	}

	install := action.NewInstall(config)

	// Configure install action
	install.ReleaseName = req.Name
	install.Namespace = req.Namespace
	install.CreateNamespace = req.CreateNamespace
	install.Wait = req.Wait
	install.WaitForJobs = req.WaitForJobs
	install.Timeout = req.Timeout
	install.SkipCRDs = req.SkipCRDs
	install.Replace = req.Replace
	install.DisableHooks = req.DisableHooks

	// Load chart
	chartPath, err := c.locateChart(req.Chart, req.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	// Parse YAML values to interface{} map
	values := make(map[string]interface{})
	if len(req.Values) > 0 {
		if err := yaml.Unmarshal([]byte(req.Values), &values); err != nil {
			return nil, fmt.Errorf("failed to parse values YAML: %w", err)
		}
	}

	// Install release
	rel, err := install.RunWithContext(ctx, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install release: %w", err)
	}

	return c.convertRelease(rel), nil
}

// UpgradeRelease upgrades an existing Helm release
func (c *helmClient) UpgradeRelease(ctx context.Context, req *UpgradeRequest) (*ReleaseInfo, error) {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(req.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create action config for namespace %s: %w", req.Namespace, err)
	}

	upgrade := action.NewUpgrade(config)

	// Configure upgrade action
	upgrade.Namespace = req.Namespace
	upgrade.Wait = req.Wait
	upgrade.WaitForJobs = req.WaitForJobs
	upgrade.Timeout = req.Timeout
	upgrade.Force = req.Force
	upgrade.ResetValues = req.ResetValues
	upgrade.ReuseValues = req.ReuseValues
	upgrade.Recreate = req.Recreate
	upgrade.MaxHistory = req.MaxHistory
	upgrade.CleanupOnFail = req.CleanupOnFail
	upgrade.DisableHooks = req.DisableHooks

	// Load chart
	chartPath, err := c.locateChart(req.Chart, req.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart: %w", err)
	}

	// Parse YAML values to interface{} map
	values := make(map[string]interface{})
	if len(req.Values) > 0 {
		if err := yaml.Unmarshal([]byte(req.Values), &values); err != nil {
			return nil, fmt.Errorf("failed to parse values YAML: %w", err)
		}
	}

	// Upgrade release
	rel, err := upgrade.RunWithContext(ctx, req.Name, chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade release: %w", err)
	}

	return c.convertRelease(rel), nil
}

// UninstallRelease uninstalls a Helm release
func (c *helmClient) UninstallRelease(ctx context.Context, req *UninstallRequest) error {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(req.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create action config for namespace %s: %w", req.Namespace, err)
	}

	uninstall := action.NewUninstall(config)

	// Configure uninstall action
	uninstall.Timeout = req.Timeout
	uninstall.DisableHooks = req.DisableHooks
	uninstall.KeepHistory = req.KeepHistory

	// Uninstall release
	if _, err := uninstall.Run(req.Name); err != nil {
		return fmt.Errorf("failed to uninstall release: %w", err)
	}

	return nil
}

// GetRelease returns information about a specific release
func (c *helmClient) GetRelease(ctx context.Context, name, namespace string) (*ReleaseInfo, error) {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create action config for namespace %s: %w", namespace, err)
	}

	get := action.NewGet(config)

	rel, err := get.Run(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get release: %w", err)
	}

	return c.convertRelease(rel), nil
}

// ListReleases returns all releases in a namespace
func (c *helmClient) ListReleases(ctx context.Context, namespace string) ([]*ReleaseInfo, error) {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create action config for namespace %s: %w", namespace, err)
	}

	list := action.NewList(config)

	// Configure list action
	if namespace != "" {
		list.Filter = fmt.Sprintf("^%s$", namespace)
	}

	releases, err := list.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	var releaseInfos []*ReleaseInfo
	for _, rel := range releases {
		releaseInfos = append(releaseInfos, c.convertRelease(rel))
	}

	return releaseInfos, nil
}

// GetReleaseHistory returns the history of a release
func (c *helmClient) GetReleaseHistory(ctx context.Context, name, namespace string) ([]*ReleaseInfo, error) {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create action config for namespace %s: %w", namespace, err)
	}

	history := action.NewHistory(config)

	releases, err := history.Run(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get release history: %w", err)
	}

	var releaseInfos []*ReleaseInfo
	for _, rel := range releases {
		releaseInfos = append(releaseInfos, c.convertRelease(rel))
	}

	return releaseInfos, nil
}

// locateChart locates a chart either locally or from a repository
func (c *helmClient) locateChart(chartRef, version string) (string, error) {
	// Create a chart downloader
	var out io.Writer = os.Stdout
	dl := downloader.ChartDownloader{
		Out:     out,
		Keyring: c.settings.RepositoryConfig,
		Getters: getter.All(c.settings),
		Options: []getter.Option{
			getter.WithBasicAuth("", ""),
		},
		RepositoryConfig: c.settings.RepositoryConfig,
		RepositoryCache:  c.settings.RepositoryCache,
	}

	// Download or locate the chart
	chartPath, _, err := dl.DownloadTo(chartRef, version, c.settings.RepositoryCache)
	if err != nil {
		return "", fmt.Errorf("failed to download chart: %w", err)
	}

	return chartPath, nil
}

// convertRelease converts a Helm release to our ReleaseInfo struct
func (c *helmClient) convertRelease(rel *release.Release) *ReleaseInfo {
	info := &ReleaseInfo{
		Name:        rel.Name,
		Namespace:   rel.Namespace,
		Revision:    rel.Version,
		Status:      rel.Info.Status.String(),
		Description: rel.Info.Description,
		Notes:       rel.Info.Notes,
	}

	// Set chart information
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		info.Chart = fmt.Sprintf("%s-%s", rel.Chart.Metadata.Name, rel.Chart.Metadata.Version)
		info.AppVersion = rel.Chart.Metadata.AppVersion
	}

	// Set timestamps
	if !rel.Info.FirstDeployed.IsZero() {
		t := rel.Info.FirstDeployed.Time
		info.FirstDeployed = &t
	}
	if !rel.Info.LastDeployed.IsZero() {
		t := rel.Info.LastDeployed.Time
		info.LastDeployed = &t
		info.Updated = t
	}

	// Set values - convert interface{} map to YAML string
	if rel.Config != nil {
		valuesYAML, err := yaml.Marshal(rel.Config)
		if err == nil {
			info.Values = string(valuesYAML)
		}
	}

	// Get original values from chart
	if rel.Chart != nil && rel.Chart.Values != nil {
		originalValuesYAML, err := yaml.Marshal(rel.Chart.Values)
		if err == nil {
			info.OriginalValues = string(originalValuesYAML)
		}
	}

	return info
}

// GetReleaseValues returns the values for a specific release
func (c *helmClient) GetReleaseValues(ctx context.Context, name, namespace string) (string, error) {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(namespace)
	if err != nil {
		return "", fmt.Errorf("failed to create action config for namespace %s: %w", namespace, err)
	}

	getValues := action.NewGetValues(config)

	values, err := getValues.Run(name)
	if err != nil {
		return "", fmt.Errorf("failed to get release values: %w", err)
	}

	valuesYAML, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to marshal values to YAML: %w", err)
	}

	return string(valuesYAML), nil
}

// GetReleaseManifest returns the manifest for a specific release
func (c *helmClient) GetReleaseManifest(ctx context.Context, name, namespace string) (string, error) {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(namespace)
	if err != nil {
		return "", fmt.Errorf("failed to create action config for namespace %s: %w", namespace, err)
	}

	get := action.NewGet(config)

	rel, err := get.Run(name)
	if err != nil {
		return "", fmt.Errorf("failed to get release: %w", err)
	}

	return rel.Manifest, nil
}

// RollbackRelease rolls back a release to a previous revision (simple version)
func (c *helmClient) RollbackRelease(ctx context.Context, name, namespace string, revision int) (*ReleaseInfo, error) {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create action config for namespace %s: %w", namespace, err)
	}

	rollback := action.NewRollback(config)
	rollback.Version = revision

	if err := rollback.Run(name); err != nil {
		return nil, fmt.Errorf("failed to rollback release: %w", err)
	}

	// Get the updated release info
	return c.GetRelease(ctx, name, namespace)
}

// TestRelease runs tests for a release
func (c *helmClient) TestRelease(ctx context.Context, name, namespace string, timeout time.Duration) error {
	// Create action configuration for the target namespace
	config, err := c.getActionConfig(namespace)
	if err != nil {
		return fmt.Errorf("failed to create action config for namespace %s: %w", namespace, err)
	}

	test := action.NewReleaseTesting(config)
	test.Timeout = timeout

	if _, err := test.Run(name); err != nil {
		return fmt.Errorf("failed to test release: %w", err)
	}

	return nil
}
