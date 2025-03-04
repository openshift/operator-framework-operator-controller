# Get the directory where this Makefile is, so we can use it below for including
DIR := $(strip $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST)))))

include $(addprefix $(DIR)/vendor/github.com/openshift/build-machinery-go/make/, \
    targets/openshift/deps.mk \
)

include $(DIR)/.bingo/Variables.mk

.PHONY: verify
verify: ## Run downstream-specific verify
	$(MAKE) tidy fmt generate -C $(DIR)/../
	$(MAKE) manifests
	git diff --exit-code

.PHONY: manifests
manifests: $(KUSTOMIZE) $(YQ)
	$(DIR)/operator-controller/generate-manifests.sh
	$(DIR)/catalogd/generate-manifests.sh
	make update-k8s-values

# Minor Kubernetes version to build against derived from the client-go dependency version
KUBE_MINOR ?= $(shell cd $(DIR)/.. && GOFLAGS=-mod=mod go list -m k8s.io/client-go | cut -d" " -f2 | sed -E 's/^v0\.([0-9]+)\.[0-9]+.*$$/1.\1/')

.PHONY: update-k8s-values # HELP Update PSA labels in config manifests with Kubernetes version
UPDATE_FILES := $(DIR)/catalogd/kustomize $(DIR)/catalogd/manifests \
               $(DIR)/operator-controller/kustomize $(DIR)/operator-controller/manifests
update-k8s-values:
	# Update PSA labels with the correct Kubernetes version
	find $(UPDATE_FILES) -type f -name '*.yaml' \
		-exec sed -i.bak -E 's/(pod-security.kubernetes.io\/[a-zA-Z-]+-version:).*/\1 "v$(KUBE_MINOR)"/' {} +
	find $(UPDATE_FILES) -type f -name '*.yaml.bak' -delete

.PHONY: verify-manifests
verify-manifests: manifests
	git diff --exit-code

E2E_REGISTRY_NAME=docker-registry
E2E_REGISTRY_NAMESPACE=operator-controller-e2e
export LOCAL_REGISTRY_HOST := $(E2E_REGISTRY_NAME).$(E2E_REGISTRY_NAMESPACE).svc:5000
export CLUSTER_REGISTRY_HOST := $(E2E_REGISTRY_NAME).$(E2E_REGISTRY_NAMESPACE).svc:5000
export REG_PKG_NAME := registry-operator
export E2E_TEST_CATALOG_V1 := e2e/test-catalog:v1
export E2E_TEST_CATALOG_V2 := e2e/test-catalog:v2
export CATALOG_IMG := $(LOCAL_REGISTRY_HOST)/$(E2E_TEST_CATALOG_V1)
# Order matters here, the ".../registries.conf" entry must be last.
export DOWNSTREAM_E2E_FLAGS := -count=1 -v -skip 'TestClusterExtensionInstallReResolvesWhenNewCatalog|TestClusterExtensionInstallRegistryDynamic|TestClusterExtensionInstallRegistry/package_requires_mirror_registry_configuration_in_/etc/containers/registries.conf'
.PHONY: test-e2e
test-e2e: ## Run the e2e tests.
	$(DIR)/operator-controller/build-test-registry.sh $(E2E_REGISTRY_NAMESPACE) $(E2E_REGISTRY_NAME) $(E2E_REGISTRY_IMAGE)
	cd $(DIR)/../; \
	go test $(DOWNSTREAM_E2E_FLAGS) ./test/e2e/...;
