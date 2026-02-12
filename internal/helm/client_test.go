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
	"testing"
	"time"
)

func TestIsOCIRegistry(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{
			name: "OCI registry",
			url:  "oci://ghcr.io/myorg/charts",
			want: true,
		},
		{
			name: "HTTPS URL",
			url:  "https://charts.example.com",
			want: false,
		},
		{
			name: "HTTP URL",
			url:  "http://charts.example.com",
			want: false,
		},
		{
			name: "empty URL",
			url:  "",
			want: false,
		},
		{
			name: "short URL",
			url:  "oci://",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOCIRegistry(tt.url); got != tt.want {
				t.Errorf("isOCIRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client == nil {
		t.Error("NewClient() returned nil client")
	}

	// Verify client implements the Client interface
	_, ok := client.(Client)
	if !ok {
		t.Error("NewClient() did not return a Client interface")
	}
}

func TestChartInfo(t *testing.T) {
	chart := ChartInfo{
		Name:        "nginx",
		Version:     "1.0.0",
		AppVersion:  "1.19.0",
		Description: "NGINX web server",
		Created:     time.Now(),
	}

	if chart.Name != "nginx" {
		t.Errorf("ChartInfo.Name = %v, want nginx", chart.Name)
	}

	if chart.Version != "1.0.0" {
		t.Errorf("ChartInfo.Version = %v, want 1.0.0", chart.Version)
	}
}

func TestReleaseInfo(t *testing.T) {
	now := time.Now()
	release := ReleaseInfo{
		Name:          "test-release",
		Namespace:     "default",
		Revision:      1,
		Status:        "deployed",
		Chart:         "nginx-1.0.0",
		FirstDeployed: &now,
		LastDeployed:  &now,
	}

	if release.Name != "test-release" {
		t.Errorf("ReleaseInfo.Name = %v, want test-release", release.Name)
	}

	if release.Revision != 1 {
		t.Errorf("ReleaseInfo.Revision = %v, want 1", release.Revision)
	}

	if release.Status != "deployed" {
		t.Errorf("ReleaseInfo.Status = %v, want deployed", release.Status)
	}
}

func TestInstallRequest(t *testing.T) {
	req := &InstallRequest{
		Name:            "my-release",
		Namespace:       "default",
		Chart:           "bitnami/nginx",
		Version:         "1.0.0",
		CreateNamespace: true,
		Wait:            true,
		Timeout:         10 * time.Minute,
	}

	if req.Name != "my-release" {
		t.Errorf("InstallRequest.Name = %v, want my-release", req.Name)
	}

	if req.Timeout != 10*time.Minute {
		t.Errorf("InstallRequest.Timeout = %v, want 10m", req.Timeout)
	}
}

func TestUpgradeRequest(t *testing.T) {
	req := &UpgradeRequest{
		Name:          "my-release",
		Namespace:     "default",
		Chart:         "bitnami/nginx",
		Version:       "1.1.0",
		Wait:          true,
		CleanupOnFail: true,
		Timeout:       15 * time.Minute,
	}

	if req.Version != "1.1.0" {
		t.Errorf("UpgradeRequest.Version = %v, want 1.1.0", req.Version)
	}

	if !req.CleanupOnFail {
		t.Error("UpgradeRequest.CleanupOnFail = false, want true")
	}
}

func TestUninstallRequest(t *testing.T) {
	req := &UninstallRequest{
		Name:         "my-release",
		Namespace:    "default",
		Timeout:      5 * time.Minute,
		DisableHooks: false,
		KeepHistory:  false,
	}

	if req.Timeout != 5*time.Minute {
		t.Errorf("UninstallRequest.Timeout = %v, want 5m", req.Timeout)
	}
}

// Rollback tests removed - using simplified interface

// Mock test for rate limiter
func TestRateLimiter(t *testing.T) {
	client, err := NewClient()
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	helmClient, ok := client.(*helmClient)
	if !ok {
		t.Fatal("Failed to cast to helmClient")
	}

	ctx := context.Background()

	// Test that rate limiter doesn't block immediately
	err = helmClient.limiter.Wait(ctx)
	if err != nil {
		t.Errorf("Rate limiter Wait() error = %v, want nil", err)
	}
}
