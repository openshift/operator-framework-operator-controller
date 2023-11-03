# Get the directory where this Makefile is, so we can use it below for including
DIR := $(strip $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST)))))

include $(addprefix $(DIR)/vendor/github.com/openshift/build-machinery-go/make/, \
    targets/openshift/deps.mk \
)

include $(DIR)/.bingo/Variables.mk

.PHONY: manifests
manifests: $(KUSTOMIZE) $(YQ)
	$(DIR)/generate-manifests.sh

.PHONY: verify-manifests
verify-manifests: manifests
	git diff --exit-code
