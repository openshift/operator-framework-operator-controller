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

export DOWNSTREAM_EXPERIMENTAL_E2E_FLAGS := -count=1 -v
.PHONY: test-experimental-e2e
test-experimental-e2e: ## Run the experimental e2e tests.
	$(DIR)/operator-controller/build-test-registry.sh $(E2E_REGISTRY_NAMESPACE) $(E2E_REGISTRY_NAME) $(E2E_REGISTRY_IMAGE)
	cd $(DIR)/../; \
	go test $(DOWNSTREAM_EXPERIMENTAL_E2E_FLAGS) ./test/experimental-e2e/...;
