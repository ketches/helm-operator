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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	helmoperatorv1alpha1 "github.com/ketches/helm-operator/api/v1alpha1"
)

var _ = Describe("HelmRelease Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-nginx"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		helmrelease := &helmoperatorv1alpha1.HelmRelease{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind HelmRelease")
			err := k8sClient.Get(ctx, typeNamespacedName, helmrelease)
			if err != nil && errors.IsNotFound(err) {
				resource := &helmoperatorv1alpha1.HelmRelease{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: helmoperatorv1alpha1.HelmReleaseSpec{
						Chart: helmoperatorv1alpha1.ChartSpec{
							Name:    "nginx",
							Version: "0.1.0",
							Repository: &helmoperatorv1alpha1.RepositoryReference{
								Name:      "test-helm-operator-charts",
								Namespace: "default",
							},
						},
						Release: &helmoperatorv1alpha1.ReleaseSpec{
							Name:      "test-nginx",
							Namespace: "default",
						},
						Values: `replicaCount: 1`,
						Install: &helmoperatorv1alpha1.InstallSpec{
							Timeout:     "10m",
							Wait:        true,
							WaitForJobs: true,
						},
						Upgrade: &helmoperatorv1alpha1.UpgradeSpec{
							Timeout:       "10m",
							Wait:          true,
							CleanupOnFail: true,
						},
						Interval: "30m",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &helmoperatorv1alpha1.HelmRelease{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance HelmRelease")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &HelmReleaseReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
