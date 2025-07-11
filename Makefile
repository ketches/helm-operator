# Image URL to use all building/pushing image targets
IMG ?= ketches/helm-operator
TAG ?= v0.1.0
ALIYUN_REGISTRY ?= registry.cn-hangzhou.aliyuncs.com

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# CONTAINER_TOOL defines the container tool to be used for building images.
# Be aware that the target commands are only tested with Docker which is
# scaffolded by default. However, you might want to replace it to use other
# tools. (i.e. podman)
CONTAINER_TOOL ?= docker

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) crd webhook paths="./..." output:crd:artifacts:config=deploy/crds
	cp deploy/crds/*.yaml charts/helm-operator/crds/

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

# TODO(user): To use a different vendor for e2e tests, modify the setup under 'tests/e2e'.
# The default setup assumes Kind is pre-installed and builds/loads the Manager Docker image locally.
# CertManager is installed by default; skip with:
# - CERT_MANAGER_INSTALL_SKIP=true
KIND_CLUSTER ?= helm-operator-test-e2e

.PHONY: setup-test-e2e
setup-test-e2e: ## Set up a Kind cluster for e2e tests if it does not exist
	@command -v $(KIND) >/dev/null 2>&1 || { \
		echo "Kind is not installed. Please install Kind manually."; \
		exit 1; \
	}
	$(KIND) create cluster --name $(KIND_CLUSTER)

.PHONY: test-e2e
test-e2e: setup-test-e2e manifests generate fmt vet ## Run the e2e tests. Expected an isolated environment using Kind.
	KIND_CLUSTER=$(KIND_CLUSTER) go test ./test/e2e/ -v -ginkgo.v
	$(MAKE) cleanup-test-e2e

.PHONY: cleanup-test-e2e
cleanup-test-e2e: ## Tear down the Kind cluster used for e2e tests
	@$(KIND) delete cluster --name $(KIND_CLUSTER)

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-config
lint-config: golangci-lint ## Verify golangci-lint linter configuration
	$(GOLANGCI_LINT) config verify

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - be able to use docker buildx. More info: https://docs.docker.com/build/buildx/
# - have enabled BuildKit. More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image to your registry (i.e. if you do not set a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To adequately provide solutions that are compatible with multiple platforms, you should consider using this option.
PLATFORMS ?= linux/arm64,linux/amd64
.PHONY: docker-build
docker-build:
	- $(CONTAINER_TOOL) buildx create --name helm-operator-builder
	$(CONTAINER_TOOL) buildx use helm-operator-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) -t ${IMG} -t ${IMG}:${TAG} -t ${ALIYUN_REGISTRY}/${IMG} -t ${ALIYUN_REGISTRY}/${IMG}:${TAG} .

.PHONY: docker-build-local
docker-build-local:
	- $(CONTAINER_TOOL) build -t ${IMG}:${TAG} .

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUBECTL) apply -f deploy/crds
	
.PHONY: uninstall
uninstall: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUBECTL) delete -f deploy/crds --ignore-not-found=$(ignore-not-found)

.PHONY: deploy
deploy: manifests ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	- $(KUBECTL) create namespace ketches || true
	$(KUBECTL) apply -n ketches -f deploy/manifests.yaml
	
.PHONY: undeploy
undeploy: manifests ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUBECTL) delete -n ketches -f deploy/manifests.yaml --ignore-not-found=$(ignore-not-found)

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KIND ?= kind
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.18.0
#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
GOLANGCI_LINT_VERSION ?= v2.1.0


.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

##@ Release Management

.PHONY: update-version
update-version: ## Update version across all files. Usage: make update-version VERSION=0.3.0
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make update-version VERSION=0.3.0"; \
		exit 1; \
	fi
	@echo "Updating version to $(VERSION)..."
	@chmod +x scripts/update-version.sh
	@./scripts/update-version.sh $(VERSION)

.PHONY: release-prepare
release-prepare: ## Prepare for release by updating version and running checks. Usage: make release-prepare VERSION=0.3.0
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-prepare VERSION=0.3.0"; \
		exit 1; \
	fi
	@echo "Preparing release $(VERSION)..."
	@make update-version VERSION=$(VERSION)
	@make manifests
	@make test
	@make lint
	@echo "Release preparation completed for version $(VERSION)"
	@echo "Next steps:"
	@echo "1. Review changes: git diff"
	@echo "2. Commit: git add . && git commit -m 'chore: bump version to $(VERSION)'"
	@echo "3. Tag: git tag -a v$(VERSION) -m 'Release v$(VERSION)'"
	@echo "4. Push: git push origin main && git push origin v$(VERSION)"

.PHONY: release-tag
release-tag: ## Create and push git tag. Usage: make release-tag VERSION=0.3.0 MESSAGE="Release notes"
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-tag VERSION=0.3.0"; \
		exit 1; \
	fi
	@if [ -z "$(MESSAGE)" ]; then \
		MESSAGE="Release v$(VERSION)"; \
	fi
	@echo "Creating tag v$(VERSION)..."
	@git tag -a v$(VERSION) -m "$(MESSAGE)"
	@git push origin v$(VERSION)
	@echo "Tag v$(VERSION) created and pushed successfully"

.PHONY: helm-package
helm-package: ## Package the Helm chart
	@echo "Packaging Helm chart..."
	@helm package charts/helm-operator
	@echo "Helm chart packaged successfully"

.PHONY: generate-changelog
generate-changelog: ## Generate changelog between tags. Usage: make generate-changelog FROM=v0.1.0 TO=v0.2.0
	@chmod +x scripts/generate-changelog.sh
	@if [ -n "$(FROM)" ] && [ -n "$(TO)" ]; then \
		./scripts/generate-changelog.sh $(FROM) $(TO); \
	elif [ -n "$(TO)" ]; then \
		./scripts/generate-changelog.sh $(TO); \
	else \
		./scripts/generate-changelog.sh; \
	fi

.PHONY: generate-release-notes
generate-release-notes: ## Generate GitHub release notes. Usage: make generate-release-notes FROM=v0.1.0 TO=v0.2.0
	@chmod +x scripts/generate-release-notes.sh
	@if [ -n "$(FROM)" ] && [ -n "$(TO)" ]; then \
		./scripts/generate-release-notes.sh $(FROM) $(TO); \
	elif [ -n "$(TO)" ]; then \
		./scripts/generate-release-notes.sh $(TO); \
	else \
		./scripts/generate-release-notes.sh; \
	fi

.PHONY: release-notes-file
release-notes-file: ## Generate release notes and save to file. Usage: make release-notes-file VERSION=0.3.0
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-notes-file VERSION=0.3.0"; \
		exit 1; \
	fi
	@echo "Generating release notes for v$(VERSION)..."
	@chmod +x scripts/generate-release-notes.sh
	@./scripts/generate-release-notes.sh v$(VERSION) > release-notes-v$(VERSION).md
	@echo "Release notes saved to release-notes-v$(VERSION).md"

.PHONY: release-complete
release-complete: ## Complete release process. Usage: make release-complete VERSION=0.3.0
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release-complete VERSION=0.3.0"; \
		exit 1; \
	fi
	@echo "Completing release $(VERSION)..."
	@make release-prepare VERSION=$(VERSION)
	@echo "Committing version changes..."
	@git add .
	@git commit -m "chore: bump version to $(VERSION)"
	@echo "Generating release notes..."
	@make release-notes-file VERSION=$(VERSION)
	@make release-tag VERSION=$(VERSION) MESSAGE="Release v$(VERSION)"
	@make helm-package
	@echo "Release $(VERSION) completed successfully!"
	@echo "Files created:"
	@echo "- release-notes-v$(VERSION).md"
	@echo "- helm-operator-$(VERSION).tgz"
	@echo ""
	@echo "Next steps:"
	@echo "1. Build and push Docker image: make docker-build docker-push IMG=$(IMG):$(VERSION)"
	@echo "2. Create GitHub release using release-notes-v$(VERSION).md"
	@echo "3. Upload helm-operator-$(VERSION).tgz to the release"
