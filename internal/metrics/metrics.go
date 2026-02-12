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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// Repository metrics
	RepositorySyncDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "helm_repository_sync_duration_seconds",
			Help:    "Duration of repository sync operations in seconds",
			Buckets: []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120},
		},
		[]string{"repository", "namespace", "type"},
	)

	RepositorySyncTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_repository_sync_total",
			Help: "Total number of repository sync attempts",
		},
		[]string{"repository", "namespace", "type", "status"},
	)

	RepositorySyncErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_repository_sync_errors_total",
			Help: "Total number of repository sync errors",
		},
		[]string{"repository", "namespace", "error_type"},
	)

	RepositoryChartsDiscovered = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helm_repository_charts_discovered",
			Help: "Number of charts discovered in repository",
		},
		[]string{"repository", "namespace"},
	)

	// Release metrics
	ReleaseOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "helm_release_operation_duration_seconds",
			Help:    "Duration of release operations in seconds",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"release", "namespace", "operation"},
	)

	ReleaseOperationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_release_operation_total",
			Help: "Total number of release operations",
		},
		[]string{"release", "namespace", "operation", "status"},
	)

	ReleaseOperationErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_release_operation_errors_total",
			Help: "Total number of release operation errors",
		},
		[]string{"release", "namespace", "operation", "error_type"},
	)

	ReleaseRollbacksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_release_rollbacks_total",
			Help: "Total number of automatic rollbacks",
		},
		[]string{"release", "namespace", "status"},
	)

	ReleaseInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helm_release_info",
			Help: "Information about Helm releases",
		},
		[]string{"release", "namespace", "chart", "version", "status"},
	)

	// ConfigMap metrics
	ChartConfigMapsGenerated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_chart_configmaps_generated_total",
			Help: "Total number of chart ConfigMaps generated",
		},
		[]string{"repository", "namespace", "chart"},
	)

	ChartConfigMapsCleaned = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_chart_configmaps_cleaned_total",
			Help: "Total number of chart ConfigMaps cleaned up",
		},
		[]string{"repository", "namespace"},
	)

	// Controller metrics
	ReconcileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_operator_reconcile_total",
			Help: "Total number of reconciliations",
		},
		[]string{"controller", "result"},
	)

	ReconcileDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "helm_operator_reconcile_duration_seconds",
			Help:    "Duration of reconciliation loops in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 5, 10},
		},
		[]string{"controller"},
	)

	ReconcileErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helm_operator_reconcile_errors_total",
			Help: "Total number of reconciliation errors",
		},
		[]string{"controller", "error_type"},
	)
)

func init() {
	// Register all metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		RepositorySyncDuration,
		RepositorySyncTotal,
		RepositorySyncErrors,
		RepositoryChartsDiscovered,
		ReleaseOperationDuration,
		ReleaseOperationTotal,
		ReleaseOperationErrors,
		ReleaseRollbacksTotal,
		ReleaseInfo,
		ChartConfigMapsGenerated,
		ChartConfigMapsCleaned,
		ReconcileTotal,
		ReconcileDuration,
		ReconcileErrors,
	)
}
