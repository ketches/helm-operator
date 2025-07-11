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
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

// Client interface defines Helm operations
type Client interface {
	RepositoryManager
	ReleaseManager
}

// RepositoryManager defines repository operations
type RepositoryManager interface {
	AddRepository(ctx context.Context, entry *repo.Entry) error
	UpdateRepository(ctx context.Context, name string) error
	GetRepositoryIndex(ctx context.Context, name string) (*repo.IndexFile, error)
	RemoveRepository(ctx context.Context, name string) error
	ListRepositories(ctx context.Context) ([]*repo.Entry, error)
	GetChartsFromRepository(ctx context.Context, repoName string) ([]ChartInfo, error)
}

// ReleaseManager defines release operations
type ReleaseManager interface {
	InstallRelease(ctx context.Context, req *InstallRequest) (*ReleaseInfo, error)
	UpgradeRelease(ctx context.Context, req *UpgradeRequest) (*ReleaseInfo, error)
	UninstallRelease(ctx context.Context, req *UninstallRequest) error
	GetRelease(ctx context.Context, name, namespace string) (*ReleaseInfo, error)
	ListReleases(ctx context.Context, namespace string) ([]*ReleaseInfo, error)
	GetReleaseHistory(ctx context.Context, name, namespace string) ([]*ReleaseInfo, error)
}

// helmClient implements the Client interface
type helmClient struct {
	settings  *cli.EnvSettings
	repoCache *repositoryCache
}

// repositoryCache manages cached repository data
type repositoryCache struct {
	mu           sync.RWMutex
	repositories []*repo.Entry
	loaded       bool
}

// NewClient creates a new Helm client
func NewClient() (Client, error) {
	settings := cli.New()

	if err := ensureRepoFile(settings); err != nil {
		return nil, fmt.Errorf("failed to ensure repository file: %w", err)
	}

	return &helmClient{
		settings:  settings,
		repoCache: &repositoryCache{},
	}, nil
}

// ensureRepoFile ensures the Helm repository file exists and is initialized
func ensureRepoFile(settings *cli.EnvSettings) error {
	// Get the repository file path
	repoFile := settings.RepositoryConfig

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(repoFile), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create repository config directory: %w", err)
	} else {
		if err := os.WriteFile(repoFile, []byte{}, 0644); err != nil {
			return fmt.Errorf("failed to create repository config file: %w", err)
		}
	}

	return nil
}

// NewClientWithSettings creates a new Helm client with custom settings
func NewClientWithSettings(settings *cli.EnvSettings) (Client, error) {
	return &helmClient{
		settings:  settings,
		repoCache: &repositoryCache{},
	}, nil
}

func debugLog(settings *cli.EnvSettings) action.DebugLog {
	if settings.Debug {
		return func(format string, v ...any) {
			format = fmt.Sprintf("[debug] %s\n", format)
			log.Output(2, fmt.Sprintf(format, v...))
		}
	}
	return func(format string, v ...interface{}) {}
}

// getActionConfig creates a new action configuration for the specified namespace
func (c *helmClient) getActionConfig(namespace string) (*action.Configuration, error) {
	config := new(action.Configuration)

	// Initialize with proper storage driver and debug function
	if err := config.Init(c.settings.RESTClientGetter(), namespace, "secrets", debugLog(c.settings)); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm configuration for namespace %s: %w", namespace, err)
	}

	// Validate that the configuration was properly initialized
	if config.KubeClient == nil {
		return nil, fmt.Errorf("kubernetes client not initialized in Helm configuration for namespace %s", namespace)
	}

	return config, nil
}

// InstallRequest contains parameters for installing a release
type InstallRequest struct {
	Name            string
	Namespace       string
	Chart           string
	Version         string
	Values          string
	CreateNamespace bool
	Wait            bool
	WaitForJobs     bool
	Timeout         time.Duration
	SkipCRDs        bool
	Replace         bool
	DisableHooks    bool
}

// UpgradeRequest contains parameters for upgrading a release
type UpgradeRequest struct {
	Name          string
	Namespace     string
	Chart         string
	Version       string
	Values        string
	Wait          bool
	WaitForJobs   bool
	Timeout       time.Duration
	Force         bool
	ResetValues   bool
	ReuseValues   bool
	Recreate      bool
	MaxHistory    int
	CleanupOnFail bool
	DisableHooks  bool
}

// UninstallRequest contains parameters for uninstalling a release
type UninstallRequest struct {
	Name         string
	Namespace    string
	Timeout      time.Duration
	DisableHooks bool
	KeepHistory  bool
}

// ReleaseInfo contains information about a release
type ReleaseInfo struct {
	Name          string
	Namespace     string
	Revision      int
	Status        string
	Chart         string
	AppVersion    string
	Updated       time.Time
	Description   string
	FirstDeployed *time.Time
	LastDeployed  *time.Time
	Notes         string
	Values        string
	RawValues     string // Default values from the chart
}

// ChartInfo contains information about a chart
type ChartInfo struct {
	Name        string
	Version     string
	AppVersion  string
	Description string
	Home        string
	Sources     []string
	Keywords    []string
	Maintainers []string
	Icon        string
	APIVersion  string
	Condition   string
	Tags        string
	Kubeversion string
	Deprecated  bool
	Annotations map[string]string
	Created     time.Time
	Digest      string
	URLs        []string
}

// RepositoryInfo contains information about a repository
type RepositoryInfo struct {
	Name                  string
	URL                   string
	Username              string
	Password              string
	CertFile              string
	KeyFile               string
	CAFile                string
	InsecureSkipTLSverify bool
}
