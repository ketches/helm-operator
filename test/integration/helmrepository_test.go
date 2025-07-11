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

package integration

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	helmoperatorv1alpha1 "github.com/ketches/helm-operator/api/v1alpha1"
	"github.com/ketches/helm-operator/internal/controller"
	"github.com/ketches/helm-operator/internal/helm"
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestHelmRepositoryIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "HelmRepository Integration Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = helmoperatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Start the manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	// Create Helm client
	helmClient, err := helm.NewClient("default")
	Expect(err).ToNot(HaveOccurred())

	// Setup HelmRepository controller
	err = (&controller.HelmRepositoryReconciler{
		Client:     mgr.GetClient(),
		Log:        ctrl.Log.WithName("controllers").WithName("HelmRepository"),
		Scheme:     mgr.GetScheme(),
		Recorder:   mgr.GetEventRecorderFor("helmrepository-controller"),
		HelmClient: helmClient,
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("HelmRepository Controller Integration", func() {
	Context("Basic Repository Operations", func() {
		It("Should sync a public repository successfully", func() {
			By("Creating a HelmRepository for Bitnami")
			repo := &helmoperatorv1alpha1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bitnami-integration",
					Namespace: "default",
				},
				Spec: helmoperatorv1alpha1.HelmRepositorySpec{
					URL:      "https://charts.bitnami.com/bitnami",
					Interval: "1h",
					Timeout:  "10m",
				},
			}

			Expect(k8sClient.Create(ctx, repo)).Should(Succeed())

			By("Waiting for repository to sync")
			repoKey := types.NamespacedName{Name: "bitnami-integration", Namespace: "default"}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Minute*3, time.Second*10).Should(BeTrue())

			By("Verifying repository status")
			Expect(len(repo.Status.Charts)).Should(BeNumerically(">", 10))
			Expect(repo.Status.Stats).ShouldNot(BeNil())
			Expect(repo.Status.Stats.TotalCharts).Should(BeNumerically(">", 10))
			Expect(repo.Status.Stats.TotalVersions).Should(BeNumerically(">", 10))
			Expect(repo.Status.LastSyncTime).ShouldNot(BeNil())

			By("Verifying chart information")
			nginxFound := false
			for _, chart := range repo.Status.Charts {
				if chart.Name == "nginx" {
					nginxFound = true
					Expect(len(chart.Versions)).Should(BeNumerically(">", 0))
					Expect(chart.Versions[0].Version).ShouldNot(BeEmpty())
					break
				}
			}
			Expect(nginxFound).Should(BeTrue(), "nginx chart should be found in Bitnami repository")
		})

		It("Should handle repository with authentication", func() {
			By("Creating a secret for repository authentication")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "repo-auth-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeOpaque,
				Data: map[string][]byte{
					"username": []byte("testuser"),
					"password": []byte("testpass"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())

			By("Creating a HelmRepository with authentication")
			repo := &helmoperatorv1alpha1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "auth-repo",
					Namespace: "default",
				},
				Spec: helmoperatorv1alpha1.HelmRepositorySpec{
					URL:      "https://charts.bitnami.com/bitnami", // Using public repo for testing
					Interval: "1h",
					Auth: &helmoperatorv1alpha1.RepositoryAuth{
						Basic: &helmoperatorv1alpha1.BasicAuth{
							SecretRef: &helmoperatorv1alpha1.SecretReference{
								Name:      "repo-auth-secret",
								Namespace: "default",
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, repo)).Should(Succeed())

			By("Waiting for repository to sync")
			repoKey := types.NamespacedName{Name: "auth-repo", Namespace: "default"}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Minute*3, time.Second*10).Should(BeTrue())

			By("Verifying repository synced successfully")
			Expect(len(repo.Status.Charts)).Should(BeNumerically(">", 0))
		})

		It("Should handle invalid repository URL gracefully", func() {
			By("Creating a HelmRepository with invalid URL")
			repo := &helmoperatorv1alpha1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-url-repo",
					Namespace: "default",
				},
				Spec: helmoperatorv1alpha1.HelmRepositorySpec{
					URL:      "https://this-is-definitely-not-a-valid-helm-repo.invalid",
					Interval: "1h",
					Timeout:  "30s",
				},
			}

			Expect(k8sClient.Create(ctx, repo)).Should(Succeed())

			By("Waiting for repository to fail")
			repoKey := types.NamespacedName{Name: "invalid-url-repo", Namespace: "default"}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Failed" && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Minute*2, time.Second*5).Should(BeTrue())

			By("Verifying error condition")
			var failedCondition *metav1.Condition
			for _, condition := range repo.Status.Conditions {
				if condition.Type == "Failed" {
					failedCondition = &condition
					break
				}
			}
			Expect(failedCondition).ShouldNot(BeNil())
			Expect(failedCondition.Message).Should(ContainSubstring("Failed to"))
		})
	})

	Context("Repository Lifecycle", func() {
		It("Should handle repository updates", func() {
			By("Creating a HelmRepository")
			repo := &helmoperatorv1alpha1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "update-test-repo",
					Namespace: "default",
				},
				Spec: helmoperatorv1alpha1.HelmRepositorySpec{
					URL:      "https://charts.bitnami.com/bitnami",
					Interval: "2h",
					Timeout:  "5m",
				},
			}

			Expect(k8sClient.Create(ctx, repo)).Should(Succeed())

			By("Waiting for initial sync")
			repoKey := types.NamespacedName{Name: "update-test-repo", Namespace: "default"}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Minute*3, time.Second*10).Should(BeTrue())

			initialGeneration := repo.Generation
			initialObservedGeneration := repo.Status.ObservedGeneration

			By("Updating repository interval")
			repo.Spec.Interval = "30m"
			Expect(k8sClient.Update(ctx, repo)).Should(Succeed())

			By("Waiting for update to be processed")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				return repo.Status.ObservedGeneration > initialObservedGeneration
			}, time.Minute*2, time.Second*5).Should(BeTrue())

			By("Verifying generation was updated")
			Expect(repo.Generation).Should(BeNumerically(">", initialGeneration))
			Expect(repo.Status.ObservedGeneration).Should(Equal(repo.Generation))
		})

		It("Should handle repository suspension", func() {
			By("Creating a HelmRepository")
			repo := &helmoperatorv1alpha1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "suspend-test-repo",
					Namespace: "default",
				},
				Spec: helmoperatorv1alpha1.HelmRepositorySpec{
					URL:      "https://charts.bitnami.com/bitnami",
					Interval: "1h",
					Suspend:  false,
				},
			}

			Expect(k8sClient.Create(ctx, repo)).Should(Succeed())

			By("Waiting for repository to be ready")
			repoKey := types.NamespacedName{Name: "suspend-test-repo", Namespace: "default"}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Minute*3, time.Second*10).Should(BeTrue())

			By("Suspending the repository")
			repo.Spec.Suspend = true
			Expect(k8sClient.Update(ctx, repo)).Should(Succeed())

			By("Waiting for repository to be suspended")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == metav1.ConditionFalse {
						return condition.Reason == "Suspended"
					}
				}
				return false
			}, time.Minute*2, time.Second*5).Should(BeTrue())
		})

		It("Should cleanup resources on deletion", func() {
			By("Creating a HelmRepository")
			repo := &helmoperatorv1alpha1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cleanup-test-repo",
					Namespace: "default",
				},
				Spec: helmoperatorv1alpha1.HelmRepositorySpec{
					URL:      "https://charts.bitnami.com/bitnami",
					Interval: "1h",
				},
			}

			Expect(k8sClient.Create(ctx, repo)).Should(Succeed())

			By("Waiting for repository to be ready")
			repoKey := types.NamespacedName{Name: "cleanup-test-repo", Namespace: "default"}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Minute*3, time.Second*10).Should(BeTrue())

			By("Deleting the repository")
			Expect(k8sClient.Delete(ctx, repo)).Should(Succeed())

			By("Waiting for repository to be deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				return err != nil
			}, time.Minute*2, time.Second*5).Should(BeTrue())
		})
	})

	Context("Error Scenarios", func() {
		It("Should handle missing authentication secret", func() {
			By("Creating a HelmRepository with non-existent secret")
			repo := &helmoperatorv1alpha1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "missing-secret-repo",
					Namespace: "default",
				},
				Spec: helmoperatorv1alpha1.HelmRepositorySpec{
					URL:      "https://charts.bitnami.com/bitnami",
					Interval: "1h",
					Auth: &helmoperatorv1alpha1.RepositoryAuth{
						Basic: &helmoperatorv1alpha1.BasicAuth{
							SecretRef: &helmoperatorv1alpha1.SecretReference{
								Name:      "non-existent-secret",
								Namespace: "default",
							},
						},
					},
				},
			}

			Expect(k8sClient.Create(ctx, repo)).Should(Succeed())

			By("Waiting for repository to fail")
			repoKey := types.NamespacedName{Name: "missing-secret-repo", Namespace: "default"}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Failed" && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Minute*2, time.Second*5).Should(BeTrue())

			By("Verifying authentication error")
			var failedCondition *metav1.Condition
			for _, condition := range repo.Status.Conditions {
				if condition.Type == "Failed" {
					failedCondition = &condition
					break
				}
			}
			Expect(failedCondition).ShouldNot(BeNil())
			Expect(failedCondition.Reason).Should(Equal("AuthenticationFailed"))
		})

		It("Should handle malformed repository index", func() {
			By("Creating a HelmRepository pointing to invalid index")
			repo := &helmoperatorv1alpha1.HelmRepository{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "malformed-index-repo",
					Namespace: "default",
				},
				Spec: helmoperatorv1alpha1.HelmRepositorySpec{
					URL:      "https://httpbin.org/json", // Returns JSON, not Helm index
					Interval: "1h",
					Timeout:  "30s",
				},
			}

			Expect(k8sClient.Create(ctx, repo)).Should(Succeed())

			By("Waiting for repository to fail")
			repoKey := types.NamespacedName{Name: "malformed-index-repo", Namespace: "default"}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, repoKey, repo)
				if err != nil {
					return false
				}
				for _, condition := range repo.Status.Conditions {
					if condition.Type == "Failed" && condition.Status == metav1.ConditionTrue {
						return true
					}
				}
				return false
			}, time.Minute*2, time.Second*5).Should(BeTrue())
		})
	})
})
