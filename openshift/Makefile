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

.PHONY: verify-manifests
verify-manifests: manifests
	git diff --exit-code

.PHONY: test-e2e
test-e2e: ## Run the e2e tests. TODO: stub until tests are working downstream
	/bin/true 
