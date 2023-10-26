# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.8. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for kustomize variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(KUSTOMIZE)
#	@echo "Running kustomize"
#	@$(KUSTOMIZE) <flags/args..>
#
KUSTOMIZE := $(GOBIN)/kustomize-v5.0.1
$(KUSTOMIZE): $(BINGO_DIR)/kustomize.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/kustomize-v5.0.1"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=kustomize.mod -o=$(GOBIN)/kustomize-v5.0.1 "sigs.k8s.io/kustomize/kustomize/v5"

YQ := $(GOBIN)/yq-v4.34.2
$(YQ): $(BINGO_DIR)/yq.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/yq-v4.34.2"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=yq.mod -o=$(GOBIN)/yq-v4.34.2 "github.com/mikefarah/yq/v4"

