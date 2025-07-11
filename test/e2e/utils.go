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

package e2e

import (
	"context"
	"time"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	helmoperatorv1alpha1 "github.com/ketches/helm-operator/api/v1alpha1"
)

// TestHelper provides utility functions for e2e tests
type TestHelper struct {
	Client client.Client
	Ctx    context.Context
}

// NewTestHelper creates a new test helper
func NewTestHelper(client client.Client, ctx context.Context) *TestHelper {
	return &TestHelper{
		Client: client,
		Ctx:    ctx,
	}
}

// WaitForRepositoryReady waits for a HelmRepository to become ready
func (h *TestHelper) WaitForRepositoryReady(name, namespace string, timeout time.Duration) *helmoperatorv1alpha1.HelmRepository {
	repo := &helmoperatorv1alpha1.HelmRepository{}
	repoKey := types.NamespacedName{Name: name, Namespace: namespace}

	gomega.Eventually(func() bool {
		err := h.Client.Get(h.Ctx, repoKey, repo)
		if err != nil {
			return false
		}
		return h.IsRepositoryReady(repo)
	}, timeout, time.Second*5).Should(gomega.BeTrue(), "Repository should become ready")

	return repo
}

// WaitForReleaseReady waits for a HelmRelease to become ready
func (h *TestHelper) WaitForReleaseReady(name, namespace string, timeout time.Duration) *helmoperatorv1alpha1.HelmRelease {
	release := &helmoperatorv1alpha1.HelmRelease{}
	releaseKey := types.NamespacedName{Name: name, Namespace: namespace}

	gomega.Eventually(func() bool {
		err := h.Client.Get(h.Ctx, releaseKey, release)
		if err != nil {
			return false
		}
		return h.IsReleaseReady(release)
	}, timeout, time.Second*10).Should(gomega.BeTrue(), "Release should become ready")

	return release
}

// WaitForRepositoryFailed waits for a HelmRepository to fail
func (h *TestHelper) WaitForRepositoryFailed(name, namespace string, timeout time.Duration) *helmoperatorv1alpha1.HelmRepository {
	repo := &helmoperatorv1alpha1.HelmRepository{}
	repoKey := types.NamespacedName{Name: name, Namespace: namespace}

	gomega.Eventually(func() bool {
		err := h.Client.Get(h.Ctx, repoKey, repo)
		if err != nil {
			return false
		}
		return h.IsRepositoryFailed(repo)
	}, timeout, time.Second*5).Should(gomega.BeTrue(), "Repository should fail")

	return repo
}

// WaitForReleaseFailed waits for a HelmRelease to fail
func (h *TestHelper) WaitForReleaseFailed(name, namespace string, timeout time.Duration) *helmoperatorv1alpha1.HelmRelease {
	release := &helmoperatorv1alpha1.HelmRelease{}
	releaseKey := types.NamespacedName{Name: name, Namespace: namespace}

	gomega.Eventually(func() bool {
		err := h.Client.Get(h.Ctx, releaseKey, release)
		if err != nil {
			return false
		}
		return h.IsReleaseFailed(release)
	}, timeout, time.Second*5).Should(gomega.BeTrue(), "Release should fail")

	return release
}

// WaitForResourceDeleted waits for a resource to be deleted
func (h *TestHelper) WaitForResourceDeleted(obj client.Object, timeout time.Duration) {
	gomega.Eventually(func() bool {
		err := h.Client.Get(h.Ctx, client.ObjectKeyFromObject(obj), obj)
		return err != nil
	}, timeout, time.Second*5).Should(gomega.BeTrue(), "Resource should be deleted")
}

// IsRepositoryReady checks if a HelmRepository is ready
func (h *TestHelper) IsRepositoryReady(repo *helmoperatorv1alpha1.HelmRepository) bool {
	for _, condition := range repo.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsRepositoryFailed checks if a HelmRepository has failed
func (h *TestHelper) IsRepositoryFailed(repo *helmoperatorv1alpha1.HelmRepository) bool {
	for _, condition := range repo.Status.Conditions {
		if condition.Type == "Failed" && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsReleaseReady checks if a HelmRelease is ready
func (h *TestHelper) IsReleaseReady(release *helmoperatorv1alpha1.HelmRelease) bool {
	for _, condition := range release.Status.Conditions {
		if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// IsReleaseFailed checks if a HelmRelease has failed
func (h *TestHelper) IsReleaseFailed(release *helmoperatorv1alpha1.HelmRelease) bool {
	for _, condition := range release.Status.Conditions {
		if condition.Type == "Failed" && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

// CreateTestRepository creates a test HelmRepository
func (h *TestHelper) CreateTestRepository(name, namespace, url string) *helmoperatorv1alpha1.HelmRepository {
	repo := &helmoperatorv1alpha1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: helmoperatorv1alpha1.HelmRepositorySpec{
			URL:      url,
			Interval: "30m",
			Timeout:  "5m",
		},
	}

	gomega.Expect(h.Client.Create(h.Ctx, repo)).Should(gomega.Succeed())
	return repo
}

// CreateTestRelease creates a test HelmRelease
func (h *TestHelper) CreateTestRelease(name, namespace, chartName, chartVersion string, repoRef *helmoperatorv1alpha1.RepositoryReference, values []byte) *helmoperatorv1alpha1.HelmRelease {
	release := &helmoperatorv1alpha1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: helmoperatorv1alpha1.HelmReleaseSpec{
			Chart: helmoperatorv1alpha1.ChartSpec{
				Name:       chartName,
				Version:    chartVersion,
				Repository: repoRef,
			},
			Release: &helmoperatorv1alpha1.ReleaseSpec{
				Name:            name + "-release",
				Namespace:       namespace,
				CreateNamespace: true,
			},
			Values: string(values),
			Install: &helmoperatorv1alpha1.InstallSpec{
				Timeout: "10m",
				Wait:    true,
			},
			Upgrade: &helmoperatorv1alpha1.UpgradeSpec{
				Timeout: "10m",
				Wait:    true,
			},
		},
	}

	gomega.Expect(h.Client.Create(h.Ctx, release)).Should(gomega.Succeed())
	return release
}

// UpdateReleaseValues updates the values of a HelmRelease
func (h *TestHelper) UpdateReleaseValues(release *helmoperatorv1alpha1.HelmRelease, newValues []byte) {
	release.Spec.Values = string(newValues)
	gomega.Expect(h.Client.Update(h.Ctx, release)).Should(gomega.Succeed())
}

// GetReleaseRevision returns the current revision of a HelmRelease
func (h *TestHelper) GetReleaseRevision(release *helmoperatorv1alpha1.HelmRelease) int {
	if release.Status.HelmRelease != nil {
		return release.Status.HelmRelease.Revision
	}
	return 0
}

// CleanupResource deletes a resource and waits for it to be removed
func (h *TestHelper) CleanupResource(obj client.Object, timeout time.Duration) {
	_ = h.Client.Delete(h.Ctx, obj)
	h.WaitForResourceDeleted(obj, timeout)
}
