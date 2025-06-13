# Get the directory where this Makefile is, so we can use it below for including
# Include the same Bingo variables used by the project
DIR := $(strip $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST)))))
include $(DIR)/../../.bingo/Variables.mk

# Definitions for the extended tests

GO_PKG_NAME := github/operator-framework-operator-controller/openshift/extended-tests

GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GIT_TREE_STATE := $(shell if git diff --quiet; then echo clean; else echo dirty; fi)

LDFLAGS := -X '$(GO_PKG_NAME)/pkg/version.CommitFromGit=$(GIT_COMMIT)' \
           -X '$(GO_PKG_NAME)/pkg/version.BuildDate=$(BUILD_DATE)' \
           -X '$(GO_PKG_NAME)/pkg/version.GitTreeState=$(GIT_TREE_STATE)'

#SECTION General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '#SECTION' and the
# target descriptions by '#HELP' or '#EXHELP'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: #HELP something, and then pretty-format the target and help. Then,
# if there's a line with #SECTION something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php
# The extended-help target uses '#EXHELP' as the delineator.

.PHONY: help
help: #HELP Display essential help.
	@awk 'BEGIN {FS = ":[^#]*#HELP"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\n"} /^[a-zA-Z_0-9-]+:.*#HELP / { printf "  \033[36m%-17s\033[0m %s\n", $$1, $$2 } ' $(MAKEFILE_LIST)

#SECTION Tests
TOOLS_BIN_DIR := $(CURDIR)/bin
GINKGO := $(TOOLS_BIN_DIR)/ginkgo

.PHONY: install-tools
install-tools: $(GINKGO) #HELP Build vendored CLI tools

$(GINKGO): vendor/modules.txt
	go build -mod=vendor -o $(GINKGO) ./vendor/github.com/onsi/ginkgo/v2/ginkgo

.PHONY: test-extended
test-extended: install-tools $(GINKGO) #HELP Run the extended tests
	$(GINKGO) $(if $(ARTIFACT_DIR),--output-dir='$(ARTIFACT_DIR)') \
		--junit-report=junit_olm.xml ./test/...

#SECTION Development
.PHONY: verify #HELP To verify the code
verify: tidy fmt vet lint

.PHONY: tidy #HELP Run go mod tidy.
tidy:
	go mod tidy

.PHONY: fmt
fmt: #HELP Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: #HELP Run go vet against code.
	go vet ./...

.PHONY: lint
lint: $(GOLANGCI_LINT) #HELP Run golangci linter.
	$(GOLANGCI_LINT) run

.PHONY: fix-lint
fix-lint: $(GOLANGCI_LINT) #HELP Fix lint issues
	$(GOLANGCI_LINT) run --fix

.PHONY: build
build: #HELP Build the extended tests binary
	# GO_COMPLIANCE_POLICY="exempt_all" must only be used for test related binaries.
	# It prevents various FIPS compliance policies from being applied to this compilation.
	# Do not set globally.
	@mkdir -p $(TOOLS_BIN_DIR)
	GO_COMPLIANCE_POLICY="exempt_all" go build -ldflags "$(LDFLAGS)" -o $(TOOLS_BIN_DIR)/extended-tests ./cmd/...
