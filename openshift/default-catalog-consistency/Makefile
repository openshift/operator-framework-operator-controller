.PHONY: test-catalog
test-catalog: ## Run the set of tests to validate the quality of catalogs
	WHAT=catalog \
	E2E_GINKGO_OPTS="$(if $(ARTIFACT_DIR),--output-dir='$(ARTIFACT_DIR)') --junit-report junit_e2e.xml" \
	go test -count=1 -v ./test/validate/...;
