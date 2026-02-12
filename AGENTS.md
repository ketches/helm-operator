# AGENTS.md

Instructions for AI coding agents working on Helm Operator. For human-facing docs see [README.md](README.md).

## Project overview

- **What it is**: Kubernetes Operator that manages Helm repositories and releases via CRDs (`HelmRepository`, `HelmRelease`).
- **Stack**: Go 1.21+, controller-runtime, Helm v3 SDK. Supports HTTP/HTTPS and **OCI** registries (OCI recommended).
- **Layout**: `api/v1alpha1/` = CRD types; `internal/controller/` = reconcilers; `internal/helm/` = Helm client; `internal/utils/` = helpers; `cmd/main.go` = entrypoint.

## Setup commands

- Install deps: `go mod download`
- Generate code (after changing types): `make generate`
- Generate CRDs/manifests: `make manifests`
- Install CRDs into cluster: `make install`
- Run controller locally: `make run` (needs cluster in `~/.kube/config`)

## Build and test commands

- Build binary: `make build`
- Run tests: `make test` (skips e2e; needs envtest)
- Lint: `make vet` then `make lint`
- Full check before commit: `make fmt vet test` or `make build`

## Code style

- Go: follow `go fmt`; use standard Go naming and error handling.
- New types or fields in `api/v1alpha1/*_types.go`: run `make generate` and `make manifests`; update both `api/v1alpha1/zz_generated.deepcopy.go` and `deploy/crds/*.yaml` (and `charts/helm-operator/crds/`).
- Put controller logic in `internal/controller/`; Helm calls in `internal/helm/`; shared helpers in `internal/utils/`.
- Keep compatibility: new spec fields should be optional with sensible defaults.

## Testing instructions

- Unit tests: `go test ./internal/... ./api/... -v` or `make test`.
- Single package: `go test ./internal/utils/... -v` or `go test ./internal/helm/... -v`.
- Coverage: `go test ./... -cover -coverprofile=cover.out` then `go tool cover -html=cover.out`.
- After changing CRDs, ensure `make generate` and `make manifests` succeed and `make build` passes.
- Add or update tests for the code you change.

## PR / commit instructions

- Run `make fmt vet test` (or `make build`) before committing.
- Title/scope: prefer conventional style, e.g. `feat(controller): add X`, `fix(helm): correct Y`.
- When touching API types, always run `make generate` and `make manifests` and commit generated files.

## Where to look

- **CRD definitions**: `api/v1alpha1/helmrepository_types.go`, `api/v1alpha1/helmrelease_types.go`
- **Reconcilers**: `internal/controller/helmrepository_controller.go`, `internal/controller/helmrelease_controller.go`
- **Helm client**: `internal/helm/client.go`, `release.go`, `repository.go`
- **Retry/errors**: `internal/utils/retry.go`
- **Metrics**: `internal/metrics/metrics.go`
- **Samples**: `samples/`, `examples/`

## Extra context

- **OCI**: Prefer OCI for charts; use `spec.chart.ociRepository` and `HelmRepository` with `url: "oci://..."`, `type: "oci"`. See [docs/oci-repository-guide.md](docs/oci-repository-guide.md) and [samples/oci-repository.yaml](samples/oci-repository.yaml).
- **ConfigMap policy**: `valuesConfigMapPolicy: disabled` is recommended to avoid resource bloat.
- **Rollback**: Releases support `spec.rollback.enabled: true` for automatic rollback on upgrade failure.
- **Detailed architecture and refactoring**: [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md).
