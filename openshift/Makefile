# Get the directory where this Makefile is, so we can use it below for including
DIR := $(strip $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST)))))

include $(addprefix $(DIR)/vendor/github.com/openshift/build-machinery-go/make/, \
    targets/openshift/deps.mk \
)

include $(DIR)/.bingo/Variables.mk

.PHONY: verify
verify: ## Run downstream-specific verify
	$(MAKE) tidy fmt vet generate -C $(DIR)/../
	$(MAKE) manifests
	git diff --exit-code

.PHONY: manifests
manifests: $(KUSTOMIZE) $(YQ)
	$(DIR)/generate-manifests.sh
	$(MAKE) update-k8s-values

# Minor Kubernetes version to build against derived from the client-go dependency version
KUBE_MINOR ?= $(shell cd $(DIR)/.. && GOFLAGS=-mod=mod go list -m k8s.io/client-go | cut -d" " -f2 | sed -E 's/^v0\.([0-9]+)\.[0-9]+.*$$/1.\1/')

.PHONY: update-k8s-values # HELP Update PSA labels in config manifests with Kubernetes version
UPDATE_FILES := $(DIR)/kustomize $(DIR)/manifests
update-k8s-values:
	# Update PSA labels with the correct Kubernetes version
	find $(UPDATE_FILES) -type f -name '*.yaml' \
		-exec sed -i.bak -E 's/(pod-security.kubernetes.io\/[a-zA-Z-]+-version:).*/\1 "v$(KUBE_MINOR)"/' {} +
	find $(UPDATE_FILES) -type f -name '*.yaml.bak' -delete

.PHONY: verify-manifests
verify-manifests: manifests
	git diff --exit-code

.PHONY: test-e2e
test-e2e: ## Run the e2e tests. TODO: stub until tests are working downstream
	/bin/true 
