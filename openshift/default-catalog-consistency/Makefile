.PHONY: test-catalog
test-catalog: ## Run the set of tests to validate the quality of catalogs
	WHAT=catalog \
	E2E_GINKGO_OPTS="$(if $(ARTIFACT_DIR),--output-dir='$(ARTIFACT_DIR)') --junit-report junit_e2e.xml" \
	go test -count=1 -v ./test/validate/...;

## To mount insecure policy from hack/policy.yaml
.PHONY: test-catalog-insecure
test-catalog-local: ## Run the set of tests to validate the quality of catalogs using insecure policy
	POLICY_PATH=$(shell realpath hack/policy.json) make test-catalog
