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

// Finalizer names
const (
	// HelmRepositoryFinalizer is the finalizer for HelmRepository resources
	HelmRepositoryFinalizer = "helm-operator.ketches.cn/repository-finalizer"

	// HelmReleaseFinalizer is the finalizer for HelmRelease resources
	HelmReleaseFinalizer = "helm-operator.ketches.cn/release-finalizer"
)
