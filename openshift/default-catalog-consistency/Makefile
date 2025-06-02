# Get the directory where this Makefile is, so we can use it below for including
# Include the same Bingo variables used by the project
DIR := $(strip $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST)))))
include $(DIR)/../../.bingo/Variables.mk

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

# Set the Ginkgo binary path. Assumes it's installed via `go install`
GOBIN ?= $(shell go env GOBIN)
ifeq ($(GOBIN),)
  GOBIN := $(shell go env GOPATH)/bin
endif

TOOLS_BIN_DIR := $(CURDIR)/bin
GINKGO := $(TOOLS_BIN_DIR)/ginkgo

.PHONY: install-tools
install-tools: $(GINKGO) #HELP Build vendored CLI tools

$(GINKGO): vendor/modules.txt
	go build -mod=vendor -o $(GINKGO) ./vendor/github.com/onsi/ginkgo/v2/ginkgo

.PHONY: test-catalog
test-catalog: install-tools $(GINKGO) #HELP Run the set of tests to validate the quality of catalogs
	$(GINKGO) $(if $(ARTIFACT_DIR),--output-dir='$(ARTIFACT_DIR)') \
		--junit-report=junit_olm.xml ./test/validate/...


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
