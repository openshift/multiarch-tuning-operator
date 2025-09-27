ifeq ($(shell test -f .env && echo -n yes),yes)
 include .env
endif

ARTIFACT_DIR ?= ./_output

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 1.2.0

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
CHANNELS=stable
DEFAULT_CHANNEL=stable
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# We want HOME to be set for all the targets. When running in Prow pods, the HOME is set to /, which is not writable.
$(info HOME is $(HOME))
ifeq ($(shell test -w /$(HOME) && echo writable),writable)
 $(info HOME is writable)
else
 $(info HOME is not writable, setting it to /tmp/build)
 HOME := /tmp/build
 $(shell mkdir -p $(HOME))
 export HOME
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# my.domain/multiarch-tuning-operator-bundle:$VERSION and my.domain/multiarch-tuning-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= registry.ci.openshift.org/origin/multiarch-tuning-operator

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
IMG ?= registry.ci.openshift.org/origin/multiarch-tuning-operator:v1.x

#### Tool Versions ####
### TODO: NOTE: Update these values to match the versions of the K8S API when pivoting to a new version of K8S.
# https://github.com/kubernetes-sigs/kustomize/releases
KUSTOMIZE_VERSION ?= v5.6.0
# https://github.com/kubernetes-sigs/controller-tools/releases
CONTROLLER_TOOLS_VERSION ?= v0.17.2
# https://github.com/kubernetes-sigs/controller-runtime/branches
SETUP_ENVTEST_VERSION ?= release-0.20
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.32.0
# https://github.com/golangci/golangci-lint/releases
GOLINT_VERSION = v2.0.2

# TODO: We'd need an upstream builder image that includes gpgme-devel (libgpgme-dev)
BUILD_IMAGE ?= registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.23-openshift-4.19
RUNTIME_IMAGE ?= quay.io/centos/centos:stream9-minimal

NO_DOCKER ?= 0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

GO111MODULE = on
export GO111MODULE
GOFLAGS ?= -mod=vendor
export GOFLAGS

ifeq ($(DBG),1)
GOGCFLAGS ?= -gcflags=all="-N -l"
endif

ifeq ($(shell command -v podman > /dev/null 2>&1 ; echo $$? ), 0)
	ENGINE=podman
else ifeq ($(shell command -v docker > /dev/null 2>&1 ; echo $$? ), 0)
	ENGINE=docker
else
	NO_DOCKER=1
endif

FORCE_DOCKER ?= 0
ifeq ($(FORCE_DOCKER), 1)
	ENGINE=docker
endif

ifeq ($(NO_DOCKER), 1)
  DOCKER_CMD =
  IMAGE_BUILD_CMD = imagebuilder
else
  DOCKER_CMD := $(ENGINE) run --env GO111MODULE=$(GO111MODULE) --env GOFLAGS=$(GOFLAGS) --env GOLINT_VERSION=$(GOLINT_VERSION) --rm  -v "$(PWD)":/go/src/github.com/outrigger-project/multiarch-tuning-operator:Z -v "$(PWD)":/go/src/github.com/openshift/multiarch-tuning-operator:Z -w /go/src/github.com/openshift/multiarch-tuning-operator $(BUILD_IMAGE)
  IMAGE_BUILD_CMD = $(ENGINE) build
endif


.PHONY: all
all: build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
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
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(MAKE) fmt

.PHONY: fmt
fmt: ## Run go fmt against code.
	$(DOCKER_CMD) hack/go-fmt.sh ./

.PHONY: vet
vet: ## Run go vet against code.
	echo "Running go vet..."
	$(DOCKER_CMD) go vet ./...

.PHONY: lint
lint:
	GOLINT_VERSION=$(GOLINT_VERSION) $(DOCKER_CMD) hack/golangci-lint.sh

.PHONY: goimports
goimports: ## Goimports against code
	$(DOCKER_CMD) hack/goimports.sh .

.PHONY: gosec
gosec: ## Run gosec.sh script to run gosec command for all the repository source code
	$(DOCKER_CMD) hack/gosec.sh ./...

.PHONY: verify-diff
verify-diff: ## Verify that no files have changed in the versioned working tree
	$(DOCKER_CMD) hack/verify-diff.sh

.PHONY: vendor
vendor: ## Run go mod vendor
	$(DOCKER_CMD) hack/go-mod.sh

.PHONY: test
test: manifests generate envtest fmt vet goimports gosec lint unit ## Run tests.
	echo "Done"

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	CGO_ENABLED=1 go build -a -o bin/manager main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: manifests generate ## Build docker image with the manager.
	$(ENGINE) build -t ${IMG} --build-arg BUILD_IMAGE=$(BUILD_IMAGE) --build-arg RUNTIME_IMAGE=$(RUNTIME_IMAGE) .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(ENGINE) push ${IMG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64
.PHONY: docker-buildx
docker-buildx: manifests generate ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	# sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	# Disabled because we need CGO_ENABLED=1
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile \
 		--build-arg BUILD_IMAGE=$(BUILD_IMAGE) --build-arg RUNTIME_IMAGE=$(RUNTIME_IMAGE) .
	- [ -f .persistent-buildx ] || docker buildx rm project-v3-builder
	# rm Dockerfile.cross

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	cd config/manager && $(KUSTOMIZE) edit set annotation multiarch.openshift.io/image:$(IMG)
	$(KUSTOMIZE) build config/standalone | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/standalone | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary. If wrong version is installed, it will be removed before downloading.
$(KUSTOMIZE): $(LOCALBIN)
	@if test -x $(LOCALBIN)/kustomize && ! $(LOCALBIN)/kustomize version | grep -q $(KUSTOMIZE_VERSION); then \
		echo "$(LOCALBIN)/kustomize version is not expected $(KUSTOMIZE_VERSION). Removing it before installing."; \
		rm -rf $(LOCALBIN)/kustomize; \
	fi
	test -s $(LOCALBIN)/kustomize || { curl -Ss $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary. If wrong version is installed, it will be overwritten.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen && $(LOCALBIN)/controller-gen --version | grep -q $(CONTROLLER_TOOLS_VERSION) || \
	GOBIN=$(LOCALBIN) GOFLAGS='' go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) GOFLAGS='' go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(SETUP_ENVTEST_VERSION)

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	cd config/manager && $(KUSTOMIZE) edit set annotation multiarch.openshift.io/image:$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle $(BUNDLE_GEN_FLAGS)
	operator-sdk bundle validate ./bundle
	VERSION=$(VERSION) hack/patch-bundle-dockerfile.sh

.PHONY: bundle-verify
bundle-verify:
	@echo "#############################################"
	@echo "####### Preparing to verify the bundle ######"
	@echo "#############################################"
	rm -f *.clusterserviceversion.yaml.created-at
	grep -rl 'createdAt:' bundle/manifests | xargs -I {} sh -c 'grep "createdAt:" {} | cut -d\" -f2 > $$(basename {}).created-at'
	@echo "######################################################"
	@echo "###### Run the creation of the bundle manfiests ######"
	@echo "######################################################"
	@$(MAKE) bundle
	@echo "##########################################################################################################"
	@echo "###### Restoring the createdAt timestamp in the bundle/manifests/*.clusterserviceversion.yaml files ######"
	@echo "##########################################################################################################"
	# Restore the createdAt timestamp in the bundle/manifests/*.clusterserviceversion.yaml files from a file named
	# *.clusterserviceversion.yaml.created-at
	for file in *.clusterserviceversion.yaml.created-at; do \
		[[ -e "$${file}" ]] || break; \
		created_at=$$(cat $${file}); \
		# single quotes are used to prevent the removal of double quotes in $${created_at} \
		sed -i 's/createdAt: .*$$/createdAt: "'$${created_at}'"/' bundle/manifests/$${file%.created-at}; \
	done
	rm -f ./*.clusterserviceversion.yaml.created-at
	@echo "#########################################################################################################"
	@echo "#### Verifying that the bundle.konflux.Dockerfile labels are in sync with the bundle.Dockerfile ones ####"
	@echo "#########################################################################################################"
	diff <(grep ^LABEL bundle.konflux.Dockerfile | sort) <(grep ^LABEL bundle.Dockerfile | sort)
	@echo "################################################################################################"
	@echo "#### Verifying no other files changed in the working tree after the bundle generation test #####"
	@echo "################################################################################################"
	@$(MAKE) verify-diff
	@echo "########################################################################"
	@echo "#### Closing successfully the verification of the bundle generation ####"
	@echo "########################################################################"

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	$(ENGINE) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool $(ENGINE) --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

GO_JUNIT_REPORT_VERSION ?= v2.1.0

unit: manifests generate envtest
	mkdir -p ${ARTIFACT_DIR}
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
		./hack/ci-test.sh

e2e: manifests generate envtest
	mkdir -p ${ARTIFACT_DIR}
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
	SKIP_COVERAGE="true" \
	TEST_LABEL="e2e" \
		./hack/ci-test.sh

.PHONY: clean
clean:
	rm -rf ${ARTIFACT_DIR}
	rm -rf ${LOCALBIN}

version:
	VERSION=$(VERSION) ./hack/bump-version.sh

.PHONY: verify-snapshots
verify-snapshots:  ## Verify snapshots for given [SNAPSHOT=..] [VERSION=..]
	./hack/check-snapshots.sh $(filter-out $@,$(MAKECMDGOALS))