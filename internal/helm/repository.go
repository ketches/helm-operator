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
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

// AddRepository adds a new repository to the Helm configuration
func (c *helmClient) AddRepository(ctx context.Context, entry *repo.Entry) error {
	// Load existing repositories
	f, err := c.loadRepoFile()
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	// Check if repository already exists
	if f.Has(entry.Name) {
		existing := f.Get(entry.Name)
		if existing.URL != entry.URL {
			return fmt.Errorf("repository %s already exists with different URL", entry.Name)
		}
		// Repository exists with same URL, update it
		f.Update(entry)
	} else {
		// Add new repository
		f.Add(entry)
	}

	// Write the repository file
	if err := f.WriteFile(c.settings.RepositoryConfig, 0644); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	// Update cache
	c.updateRepositoryCache(f.Repositories)

	// Update the repository index
	return c.UpdateRepository(ctx, entry.Name)
}

// UpdateRepository updates the index for a specific repository
func (c *helmClient) UpdateRepository(ctx context.Context, name string) error {
	f, err := c.loadRepoFile()
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	entry := f.Get(name)
	if entry == nil {
		return fmt.Errorf("repository %s not found", name)
	}

	// Create repository cache directory
	cacheDir := c.settings.RepositoryCache
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create chart repository
	chartRepo, err := repo.NewChartRepository(entry, getter.All(c.settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repository: %w", err)
	}

	// Set cache file path
	chartRepo.CachePath = cacheDir

	// Download index file with context timeout
	indexFile, err := chartRepo.DownloadIndexFile()
	if err != nil {
		return fmt.Errorf("failed to download repository index: %w", err)
	}

	// Verify index file
	if _, err := repo.LoadIndexFile(indexFile); err != nil {
		return fmt.Errorf("invalid repository index: %w", err)
	}

	return nil
}

// GetRepositoryIndex returns the index file for a repository
func (c *helmClient) GetRepositoryIndex(ctx context.Context, name string) (*repo.IndexFile, error) {
	f, err := c.loadRepoFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	entry := f.Get(name)
	if entry == nil {
		return nil, fmt.Errorf("repository %s not found", name)
	}

	// Get index file path
	cacheDir := c.settings.RepositoryCache
	indexFile := filepath.Join(cacheDir, fmt.Sprintf("%s-index.yaml", name))

	// Load index file
	index, err := repo.LoadIndexFile(indexFile)
	if err != nil {
		// If index file doesn't exist or is invalid, try to update it
		if os.IsNotExist(err) || repo.ErrNoAPIVersion.Error() == err.Error() {
			if updateErr := c.UpdateRepository(ctx, name); updateErr != nil {
				return nil, fmt.Errorf("failed to update repository %s: %w", name, updateErr)
			}
			// Try to load again
			index, err = repo.LoadIndexFile(indexFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load repository index after update: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load repository index: %w", err)
		}
	}

	return index, nil
}

// RemoveRepository removes a repository from the Helm configuration
func (c *helmClient) RemoveRepository(ctx context.Context, name string) error {
	f, err := c.loadRepoFile()
	if err != nil {
		return fmt.Errorf("failed to load repository file: %w", err)
	}

	if !f.Has(name) {
		return fmt.Errorf("repository %s not found", name)
	}

	// Remove repository from file
	if !f.Remove(name) {
		return fmt.Errorf("failed to remove repository %s", name)
	}

	// Write the updated repository file
	if err := f.WriteFile(c.settings.RepositoryConfig, 0644); err != nil {
		return fmt.Errorf("failed to write repository file: %w", err)
	}

	// Update cache
	c.updateRepositoryCache(f.Repositories)

	// Remove cache files
	cacheDir := c.settings.RepositoryCache
	indexFile := filepath.Join(cacheDir, fmt.Sprintf("%s-index.yaml", name))
	if err := os.Remove(indexFile); err != nil && !os.IsNotExist(err) {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to remove cache file %s: %v\n", indexFile, err)
	}

	return nil
}

// ListRepositories returns all configured repositories
func (c *helmClient) ListRepositories(ctx context.Context) ([]*repo.Entry, error) {
	// Try to get from cache first
	c.repoCache.mu.RLock()
	if c.repoCache.loaded && c.repoCache.repositories != nil {
		repos := make([]*repo.Entry, len(c.repoCache.repositories))
		copy(repos, c.repoCache.repositories)
		c.repoCache.mu.RUnlock()
		return repos, nil
	}
	c.repoCache.mu.RUnlock()

	// Cache miss or nil, load from file
	f, err := c.loadRepoFile()
	if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	// Update cache
	c.repoCache.mu.Lock()
	c.repoCache.repositories = make([]*repo.Entry, len(f.Repositories))
	copy(c.repoCache.repositories, f.Repositories)
	c.repoCache.loaded = true
	c.repoCache.mu.Unlock()

	return f.Repositories, nil
}

// GetChartsFromRepository returns all charts from a repository
func (c *helmClient) GetChartsFromRepository(ctx context.Context, repoName string) ([]ChartInfo, error) {
	index, err := c.GetRepositoryIndex(ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository index: %w", err)
	}

	var charts []ChartInfo
	for chartName, chartVersions := range index.Entries {
		if len(chartVersions) == 0 {
			continue
		}

		// Get the latest version (first in the list)
		latest := chartVersions[0]

		chart := ChartInfo{
			Name:        chartName,
			Version:     latest.Version,
			AppVersion:  latest.AppVersion,
			Description: latest.Description,
			Home:        latest.Home,
			Sources:     latest.Sources,
			Keywords:    latest.Keywords,
			Icon:        latest.Icon,
			APIVersion:  latest.APIVersion,
			Condition:   latest.Condition,
			Tags:        latest.Tags,
			Kubeversion: latest.KubeVersion,
			Deprecated:  latest.Deprecated,
			Annotations: latest.Annotations,
			Created:     latest.Created,
			Digest:      latest.Digest,
			URLs:        latest.URLs,
		}

		// Add maintainers
		for _, maintainer := range latest.Maintainers {
			chart.Maintainers = append(chart.Maintainers, maintainer.Name)
		}

		charts = append(charts, chart)
	}

	return charts, nil
}

// GetChartVersions returns all versions of a specific chart from a repository
func (c *helmClient) GetChartVersions(ctx context.Context, repoName, chartName string) ([]ChartInfo, error) {
	index, err := c.GetRepositoryIndex(ctx, repoName)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository index: %w", err)
	}

	chartVersions, exists := index.Entries[chartName]
	if !exists {
		return nil, fmt.Errorf("chart %s not found in repository %s", chartName, repoName)
	}

	var versions []ChartInfo
	for _, chartVersion := range chartVersions {
		chart := ChartInfo{
			Name:        chartName,
			Version:     chartVersion.Version,
			AppVersion:  chartVersion.AppVersion,
			Description: chartVersion.Description,
			Home:        chartVersion.Home,
			Sources:     chartVersion.Sources,
			Keywords:    chartVersion.Keywords,
			Icon:        chartVersion.Icon,
			APIVersion:  chartVersion.APIVersion,
			Condition:   chartVersion.Condition,
			Tags:        chartVersion.Tags,
			Kubeversion: chartVersion.KubeVersion,
			Deprecated:  chartVersion.Deprecated,
			Annotations: chartVersion.Annotations,
			Created:     chartVersion.Created,
			Digest:      chartVersion.Digest,
			URLs:        chartVersion.URLs,
		}

		// Add maintainers
		for _, maintainer := range chartVersion.Maintainers {
			chart.Maintainers = append(chart.Maintainers, maintainer.Name)
		}

		versions = append(versions, chart)
	}

	return versions, nil
}

// loadRepoFile loads the Helm repository file
func (c *helmClient) loadRepoFile() (*repo.File, error) {
	repoFile := c.settings.RepositoryConfig

	f, err := repo.LoadFile(repoFile)
	if os.IsNotExist(err) {
		f = repo.NewFile()
	} else if err != nil {
		return nil, fmt.Errorf("failed to load repository file: %w", err)
	}

	return f, nil
}

// updateRepositoryCache updates the repository cache with the latest data
func (c *helmClient) updateRepositoryCache(repositories []*repo.Entry) {
	c.repoCache.mu.Lock()
	defer c.repoCache.mu.Unlock()

	c.repoCache.repositories = make([]*repo.Entry, len(repositories))
	copy(c.repoCache.repositories, repositories)
	c.repoCache.loaded = true
}

// GetChartValues downloads and extracts the values.yaml from a specific chart version
func (c *helmClient) GetChartValues(ctx context.Context, repoName, chartName, version string) (string, error) {
	// Get repository entry
	f, err := c.loadRepoFile()
	if err != nil {
		return "", fmt.Errorf("failed to load repository file: %w", err)
	}

	entry := f.Get(repoName)
	if entry == nil {
		return "", fmt.Errorf("repository %s not found", repoName)
	}

	// Create chart downloader
	dl := downloader.ChartDownloader{
		Out:              os.Stdout,
		Keyring:          c.settings.RepositoryConfig,
		Getters:          getter.All(c.settings),
		RepositoryConfig: c.settings.RepositoryConfig,
		RepositoryCache:  c.settings.RepositoryCache,
	}

	// Download the chart
	chartRef := fmt.Sprintf("%s/%s", repoName, chartName)
	chartPath, _, err := dl.DownloadTo(chartRef, version, c.settings.RepositoryCache)
	if err != nil {
		return "", fmt.Errorf("failed to download chart %s:%s: %w", chartRef, version, err)
	}

	// Load the chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart %s: %w", chartPath, err)
	}

	// Extract values.yaml
	var valuesContent string
	for _, file := range chart.Raw {
		if file.Name == "values.yaml" {
			valuesContent = string(file.Data)
			break
		}
	}

	// If no values.yaml found, return default values from chart metadata
	if valuesContent == "" && chart.Values != nil {
		valuesBytes, err := yaml.Marshal(chart.Values)
		if err != nil {
			return "", fmt.Errorf("failed to marshal default values: %w", err)
		}
		valuesContent = string(valuesBytes)
	}

	// Clean up downloaded chart file
	if err := os.Remove(chartPath); err != nil && !os.IsNotExist(err) {
		// Log warning but don't fail
		fmt.Printf("Warning: failed to remove downloaded chart file %s: %v\n", chartPath, err)
	}

	return valuesContent, nil
}
