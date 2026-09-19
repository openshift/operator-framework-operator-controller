## Directives

Produce concise code, comment and agent output optimized for developer experience.
Run `go fmt ./...` to format .go files, and run `./do dev:lintfix` to lint them.
Run `go mod tidy` after any go.mod changes.
Run single unit tests with `go test -run '^TestName$' ./modulepath/`.
Run the full unit test suite with `./do dev:unit`.
Run single e2e tests with `./do dev:integration <golang test filter>`.
Run the full e2e test suite with `./do dev:integration`. The test cluster MUST be removed with `./do dev:destroy` before rerunning e2e tests.
Ensure no trailing whitespace in edited files but keep a trailing newline.
Preserve existing code comments, do not remove or rewrite comments that are still relevant.
Respect when a user tells you that a relevant repo is checked out relative to this project.

## Notes

This repo makes heavy use of kubernetes' server-side apply feature.

Known users of this codebase are:
- github.com/package-operator/package-operator
- github.com/operator-framework/operator-controller
- github.com/openshift/cluster-capi-operator
