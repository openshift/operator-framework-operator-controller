# AGENTS.md - Guidance for AI Tools

This document provides essential information for AI coding assistants working with the OpenShift operator-controller repository.

---

## Critical Context: This is a Fork

This repository (`openshift/operator-framework-operator-controller`) is OpenShift's fork of upstream operator-controller (`operator-framework/operator-controller`). The maintainers work to minimize the diff between upstream and this fork. AI tools MUST understand the special constraints and workflows that govern changes to this codebase.

**Upstream Repository:** https://github.com/operator-framework/operator-controller

**For general operator-controller information**, refer to the upstream AGENTS.md file at the root of this repository, which is synced from upstream.

---

## Why This Matters for AI Tools

Unlike most repositories where you can freely suggest changes anywhere in the codebase, this fork:

- **Gets rebased regularly** against upstream operator-controller (with each upstream release)
- **Must maintain minimal divergence** from upstream
- **Has strict commit message conventions** that affect how changes survive rebases
- **Requires changes to be categorized** as either upstream cherry-picks or downstream carries

**Before suggesting any code changes, AI tools MUST understand the carry patch process documented below.**

---

## Understanding the Rebase Process

Every time a new upstream operator-controller version is released, the maintainers perform a rebase with these steps:

1. **Remove all carry patches** from the current branch
2. **Cherry-pick all new upstream commits** added between releases
3. **Reapply all carry patches** on top of the new upstream base

This means that **carry patches are a manual maintenance burden**. Each carry patch must be:

- **Manually reapplied** during every rebase
- **Reviewed for conflicts** with new upstream changes
- **Updated** if upstream code has changed in conflicting ways
- **Maintained** by the OpenShift team indefinitely

**Carry patches should only be added if a change absolutely cannot be accepted upstream.** Every carry patch represents ongoing manual work for the maintainers through every upstream release.

---

## Repository Structure

### Core operator-controller Code (Upstream)

- `api/` - API definitions and CRDs
- `cmd/` - Main binaries (operator-controller, catalogd)
- `internal/` - Internal implementation code
- `pkg/` - Public packages
- `test/` - Test suites (e2e, unit, integration)
- `helm/` - Helm charts
- `docs/` - Documentation

### OpenShift-Specific Code (Downstream)

- `openshift/` - **ALL OpenShift-specific code and configuration**
  - `openshift/Makefile` - Downstream build targets
  - `openshift/catalogd.Dockerfile` - OpenShift catalogd build
  - `openshift/operator-controller.Dockerfile` - OpenShift operator-controller build
  - `openshift/tests-extension/` - OpenShift Test Extension (OTE) tests
  - `openshift/dev_notes.md` - Downstreaming process documentation
  - `openshift/HOWTO-origin-tests.md` - Origin test instructions
  - `openshift/AGENTS.md` - This file
- `.ci-operator.yaml` - OpenShift CI configuration (replaces GitHub Actions)

---

## Commit Message Conventions (CRITICAL)

**ALL commits to this repository MUST use one of these prefixes:**

### `UPSTREAM: <carry>:`

Changes that should be reapplied in future rebases.

**When to use:** Any change that needs to persist across upstream version bumps

**Lifecycle:** Manually reapplied during every rebase

**Examples:**
```
UPSTREAM: <carry>: Add OpenShift-specific test for disconnected environments
UPSTREAM: <carry>: Update OCP catalogs to v4.21
UPSTREAM: <carry>: Use busybox/httpd to simulate probes
UPSTREAM: <carry>: Add OTE tests for OLMv1 stress scenarios
```

**IMPORTANT:** These create ongoing maintenance burden - only use when the change cannot go upstream.

### `UPSTREAM: <drop>:`

Changes that should NOT be included in future rebases.

**When to use:** Any temporary or regeneratable change specific to this branch

**Lifecycle:** Dropped during rebases

**Common cases:**
- Generated files that will be regenerated (`make manifests`)
- Vendored dependencies that will be re-vendored
- Behavior changes specific to just this release
- Branch-specific fixes that won't apply to future versions

**Examples:**
```
UPSTREAM: <drop>: Remove upstream GitHub Actions workflows
UPSTREAM: <drop>: Remove upstream codecov configuration
UPSTREAM: <drop>: make manifests
```

### `UPSTREAM: <PR number>:`

Cherry-picks from upstream operator-controller PRs.

**When to use:** Backporting an upstream fix before the next rebase

**Format:** The number is the upstream PR ID from `operator-framework/operator-controller`

**Lifecycle:** Only picked if not yet in the new upstream base

**Example:**
```
UPSTREAM: 567: Fix reconciliation loop for ClusterExtension

Cherry-picked from upstream PR operator-framework/operator-controller#567

Signed-off-by: Jane Doe <jane@example.com>
```

### Direct upstream commits (no prefix)

Commits cherry-picked directly from `operator-framework/operator-controller` with their original commit messages.

**When to use:** ONLY during the rebase process itself

**AI tools should NEVER suggest these** in regular pull requests - these are added by maintainers when rebasing to a new upstream version.

---

## Important Squashing Rules

- **OpenShift-specific files** in `openshift/` should be squashed into focused commits
- **Generated changes** must NEVER be mixed with code changes
- **Related carries** should be squashed together to simplify future rebases

---

## Enforcement

**Pull requests that do not follow these commit message conventions will be rejected by maintainers.** Every commit must use one of the `UPSTREAM:` prefixes listed above (except for direct upstream commits added during rebase).

---

## Rules for AI Tools Making Changes

### DO NOT Freely Modify Files

Unlike typical repositories, you cannot simply suggest changes anywhere. You MUST:

1. **Understand the intent:** Is this fixing an OpenShift-specific issue or an upstream bug?

2. **Choose the right approach:**
   - If it's an upstream bug → Should be fixed in `operator-framework/operator-controller` first, then cherry-picked
   - If it's OpenShift-specific → Use `UPSTREAM: <carry>:` prefix
   - If it's generated code → Will be handled by `make manifests` with `UPSTREAM: <drop>:`

3. **Use the correct commit prefix** based on the rules above

### Prefer Upstream Fixes

If you identify a bug that affects operator-controller generally (not just OpenShift):

1. **The fix should ideally go to upstream `operator-framework/operator-controller` first**
2. **Then cherry-pick** to OpenShift using `UPSTREAM: <PR number>:` format
3. **Only use `UPSTREAM: <carry>:`** for truly OpenShift-specific behavior

**Decision tree:**

| Change Type | Prefix | Upstream First? |
|------------|--------|----------------|
| General bug fix | `UPSTREAM: <PR>:` | ✅ YES - Fix upstream, then cherry-pick |
| General feature | `UPSTREAM: <PR>:` | ✅ YES - Add upstream, then cherry-pick |
| OpenShift test (OTE) | `UPSTREAM: <carry>:` | ❌ NO - Downstream only |
| OpenShift integration | `UPSTREAM: <carry>:` | ❌ NO - Downstream only |
| OpenShift config | `UPSTREAM: <carry>:` | ❌ NO - Downstream only |
| Remove upstream file | `UPSTREAM: <drop>:` | ❌ NO - Downstream only |
| Generated files | `UPSTREAM: <drop>:` | ❌ NO - Regenerate with `make manifests` |

### Generated Files Require Special Handling

These files are generated and must not be hand-edited:

- CRD manifests in `helm/olmv1/base/*/crd/`
- Generated manifests in `manifests/`
- DeepCopy methods in `api/v1/zz_generated.deepcopy.go`
- CRD reference docs in `docs/api-reference/`

**To update generated files:**

```bash
# From repository root
make manifests

# Commit with drop prefix
git commit -s -m "UPSTREAM: <drop>: make manifests"
```

### Dependency Updates

When updating Go dependencies:

```bash
# From repository root
go mod tidy

# Commit with drop prefix (if downstream-specific)
git commit -s -m "UPSTREAM: <drop>: go mod tidy"
```

---

## Build and Test Commands

AI tools should recommend these commands:

### Build

```bash
# From repository root
make build

# From openshift directory (downstream-specific)
cd openshift
make verify
```

### Run Tests

```bash
# Unit tests
make test-unit

# E2E tests
make test-e2e

# Downstream E2E tests (from openshift/)
cd openshift
make test-e2e
make test-experimental-e2e
```

### Verify Code Quality

```bash
# From repository root
make verify

# From openshift directory
cd openshift
make verify
```

This runs linting, generated file checks, and other verification.

### Update Generated Files

```bash
# From repository root
make manifests
make generate
```

---

## Key Directories to Avoid Modifying Carelessly

### High-Risk Areas (Upstream Code)

Changes here create rebase burden and should be minimized:

- `api/` - API definitions
- `internal/` - Internal implementation
- `cmd/` - Main binaries
- `pkg/` - Public packages
- `test/` - Test suites
- `helm/` - Helm charts

**If changes are needed here, strongly consider:**

1. Can this be fixed upstream first?
2. Is there a way to achieve this with less invasive changes?
3. Can this be done via OpenShift-specific configuration in `openshift/`?

### Safe Areas (Downstream Code)

Changes here are expected:

- `openshift/` - All OpenShift-specific code
- `openshift/Makefile` - Downstream build targets
- `openshift/tests-extension/` - OTE tests
- `openshift/*.Dockerfile` - OpenShift container builds
- OpenShift-specific documentation in `openshift/`

---

## Best Practices for AI Tools

1. **Read before writing:** Always examine existing code and commit history before suggesting changes
2. **Understand the context:** Is this addressing an upstream or downstream concern?
3. **Minimize divergence:** Prefer smaller, more targeted changes
4. **Use correct prefixes:** Every commit must have an `UPSTREAM:` prefix
5. **Check for upstream fixes:** Before implementing a fix, search if it exists upstream
6. **Suggest testing:** Always recommend running tests (`make test`, `make verify`)
7. **Consider rebase impact:** How will this change survive future rebases?
8. **Prefer `openshift/` for new OpenShift code:** Keep downstream changes in the `openshift/` directory

---

## Common Pitfalls to Avoid

- ❌ Mixing generated and code changes in the same commit
- ❌ Using wrong `UPSTREAM:` prefix (causes rebase issues)
- ❌ Modifying core operator-controller code without understanding if it should be upstream
- ❌ Skipping `make manifests` after API or dependency changes
- ❌ Not running `make verify` before suggesting the change is complete
- ❌ Suggesting changes to generated files (should use `make manifests`)
- ❌ Forgetting DCO sign-off (`git commit -s`)

---

## Examples of Good vs. Bad Suggestions

### ❌ Bad: Suggesting direct changes to core code without context

**User:** "The ClusterExtension reconciliation is broken"

**AI:** "I'll fix this bug in `internal/controllers/clusterextension_controller.go` by changing line 123..."

**Problem:** Doesn't consider if this should be an upstream fix.

### ✅ Good: Understanding the context first

**User:** "The ClusterExtension reconciliation is broken"

**AI:** "This appears to be a general operator-controller issue. I recommend:

1. Check if this is already fixed upstream in `operator-framework/operator-controller`
2. If not, consider opening an upstream PR first
3. If OpenShift needs it urgently, cherry-pick with: `UPSTREAM: <PR>: <description>`
4. If it's truly OpenShift-specific (e.g., disconnected environments), use: `UPSTREAM: <carry>: <description>`

Can you confirm if this issue also affects upstream operator-controller?"

---

### ❌ Bad: Mixing concerns in one commit

**Commit:**
```
UPSTREAM: <carry>: Fix authentication and update manifests

Changes:
- Modified internal/auth/openshift.go
- Regenerated CRD manifests
- Updated go.mod
```

**Problem:** Mixes code changes with generated file updates.

### ✅ Good: Separate commits

**Commit 1:**
```
UPSTREAM: <carry>: Fix OpenShift authentication integration

Changes:
- Modified internal/auth/openshift.go

Signed-off-by: Jane Doe <jane@example.com>
```

**Commit 2:**
```
UPSTREAM: <drop>: make manifests

Changes:
- Regenerated CRD manifests

Signed-off-by: Jane Doe <jane@example.com>
```

**Commit 3:**
```
UPSTREAM: <drop>: go mod tidy

Changes:
- Updated go.mod and go.sum

Signed-off-by: Jane Doe <jane@example.com>
```

---

### ❌ Bad: OpenShift test in upstream code

**AI:** "I'll add a test for disconnected environments in `test/e2e/`..."

**Problem:** OpenShift-specific tests should go in `openshift/tests-extension/` (OTE).

### ✅ Good: OpenShift test in correct location

**AI:** "This is an OpenShift-specific test for disconnected environments. I'll add it to `openshift/tests-extension/` and commit with:

```
UPSTREAM: <carry>: Add OTE test for disconnected installation

This test validates OLMv1 behavior in OpenShift disconnected
environments with mirrored catalogs.

Signed-off-by: Your Name <your@example.com>
```

Then run `cd openshift && make test-e2e` to verify."

---

## Questions AI Tools Should Ask

When a user requests a change, consider:

1. **Is this fixing an operator-controller bug or adding OpenShift-specific behavior?**
2. **Does this change already exist in a newer version of upstream operator-controller?**
3. **Will this change need to survive future rebases?**
4. **Are there generated files that need updating after this change?**
5. **Does this change require updates to dependencies?**
6. **What tests should be added or updated?**
7. **Should this go in the `openshift/` directory or in upstream code?**
8. **Is there a less invasive way to achieve this?**

---

## OpenShift-Specific Directory Structure

The `openshift/` directory contains all downstream-specific code, configuration, and tooling:

```
openshift/
├── Makefile                          # Downstream build targets
├── .bingo/                           # Downstream tool dependencies
├── catalogd/                         # OpenShift catalogd configurations
│   └── build-test-registry.sh       # Test registry setup script
├── catalogd.Dockerfile               # OpenShift catalogd container build
├── operator-controller/              # OpenShift operator-controller configs
│   └── build-test-registry.sh       # Test registry setup script
├── operator-controller.Dockerfile    # OpenShift operator-controller build
├── registry.Dockerfile               # OpenShift registry image build
├── helm/                             # OpenShift Helm chart modifications
├── tests-extension/                  # OpenShift Test Extension (OTE) tests
│   └── ...                          # OTE test files
├── default-catalog-consistency/      # Default catalog consistency checker
├── vendor/                           # Vendored dependencies for openshift/
├── dev_notes.md                      # Downstreaming process documentation
├── HOWTO-origin-tests.md            # Instructions for running origin tests
└── AGENTS.md                         # This file (downstream-specific guide)
```

---

## Common OpenShift-Specific Tasks

### Adding OpenShift-Specific Tests

**OpenShift Test Extension (OTE) tests** go in `openshift/tests-extension/`:

```bash
# 1. Create test file in openshift/tests-extension/
# 2. Follow OTE test patterns
# 3. Commit with UPSTREAM: <carry>: prefix

git commit -s -m "UPSTREAM: <carry>: Add OTE test for disconnected installation

This test validates OLMv1 behavior in OpenShift disconnected
environments with mirrored catalogs.

Signed-off-by: Your Name <your@example.com>"

# 4. Test it
cd openshift
make test-e2e
```

### Updating OpenShift Catalogs

```bash
# Make changes to catalog configurations
# ...

# Commit with carry prefix
git commit -s -m "UPSTREAM: <carry>: Update OCP catalogs to v4.22

Updates catalog references to OpenShift 4.22 release catalogs.

Signed-off-by: Your Name <your@example.com>"
```

### Modifying Downstream Build

Changes to `openshift/Makefile` or Dockerfiles:

```bash
# Make changes
# ...

# Commit with carry prefix
git commit -s -m "UPSTREAM: <carry>: Update operator-controller Dockerfile for RHEL 9

Adjusts base image and build process for RHEL 9 compatibility.

Signed-off-by: Your Name <your@example.com>"
```

### Dropping Upstream Configuration

Removing upstream files that don't apply to OpenShift:

```bash
# Remove files
git rm .github/workflows/some-workflow.yaml

# Commit with drop prefix
git commit -s -m "UPSTREAM: <drop>: Remove upstream GitHub Actions workflow

OpenShift uses Prow CI instead of GitHub Actions.

Signed-off-by: Your Name <your@example.com>"
```

---

## Running OpenShift Origin Tests

OpenShift includes integration tests in the `openshift/origin` repository that validate OLMv1.

### Setup

1. **Clone the origin repository:**
   ```bash
   git clone https://github.com/openshift/origin.git
   cd origin
   ```

2. **Build the test binary:**
   ```bash
   make WHAT=cmd/openshift-tests
   ```

3. **Set up cluster access:**
   ```bash
   export KUBECONFIG=/path/to/kubeconfig
   ```

### Running OLMv1 Tests

```bash
# List all OLMv1 tests
./openshift-tests run all --dry-run | grep sig-olmv1

# Run all OLMv1 tests
./openshift-tests run all --dry-run | grep sig-olmv1 | ./openshift-tests run -f -
```

For complete instructions, see `openshift/HOWTO-origin-tests.md`.

---

## OpenShift CI Configuration

The `.ci-operator.yaml` file at the repository root configures OpenShift CI:

```yaml
build_root_image:
  name: release
  namespace: openshift
  tag: rhel-9-release-golang-1.24-openshift-4.21
```

**Key differences from upstream:**

| Aspect | Upstream | Downstream |
|--------|----------|------------|
| CI System | GitHub Actions | OpenShift CI (Prow) |
| Configuration | `.github/workflows/` | `.ci-operator.yaml` |
| Build Image | Ubuntu-based | RHEL-based |
| Test Environment | GitHub runners | OpenShift CI clusters |

---

## Downstream Rebase Process

When syncing with a new upstream release:

1. **Identify carry commits:**
   ```bash
   git log --oneline --grep="UPSTREAM: <carry>"
   git log --oneline --grep="UPSTREAM: <drop>"
   ```

2. **Rebase on new upstream:**
   ```bash
   git fetch upstream
   git rebase upstream/main  # or upstream/v1.x.y
   ```

3. **Resolve conflicts:**
   - Conflicts often occur in carry commits
   - Update carry commits if upstream code changed significantly
   - Test thoroughly after resolving conflicts

4. **Verify after rebase:**
   ```bash
   cd openshift
   make verify
   make test-e2e
   ```

**To simplify future rebases, squash related carry commits together.**

---

## Testing Your Changes

Before suggesting a change is complete, ensure:

```bash
# From repository root
make build          # Build succeeds
make test-unit      # Unit tests pass
make verify         # Verification passes

# From openshift directory
cd openshift
make verify         # Downstream verification passes
make test-e2e       # Downstream e2e tests pass
```

---

## Resources for AI Tools

- **Upstream AGENTS.md:** Root of this repository (synced from upstream)
- **Upstream documentation:** https://operator-framework.github.io/operator-controller/
- **Downstreaming guide:** `openshift/dev_notes.md`
- **Origin tests guide:** `openshift/HOWTO-origin-tests.md`
- **Downstreaming tooling:** https://github.com/openshift/operator-framework-tooling

---

## Summary for AI Tools

**The most important thing to remember:** This is not a normal repository. It's a carefully maintained fork that gets regularly rebased. Every change must be made with the understanding of:

1. **Whether it belongs upstream or downstream**
2. **How it will survive future rebases**
3. **What commit message prefix it requires**
4. **The ongoing maintenance burden of carry patches**

**When in doubt:**
- Ask the user whether this is OpenShift-specific or upstream
- Look at recent commit history for similar changes
- Prefer upstream fixes over downstream carries
- Minimize code divergence
- Always use correct `UPSTREAM:` prefixes

---

## Quick Reference Commands

```bash
# Downstream build and verify
cd openshift
make verify                    # Run all verification
make manifests                 # Generate manifests
make test-e2e                  # Run e2e tests
make test-experimental-e2e     # Run experimental e2e tests

# Check commit formatting
git log --oneline -20 --grep="UPSTREAM"
git log --oneline --grep="UPSTREAM: <carry>"
git log --oneline --grep="UPSTREAM: <drop>"

# Proper commit format
git commit -s -m "UPSTREAM: <carry>: Description

Detailed explanation.

Signed-off-by: Your Name <your@example.com>"

# Run origin tests (from openshift/origin repo)
./openshift-tests run all --dry-run | grep sig-olmv1 | ./openshift-tests run -f -
```

---

## DCO Requirement

All commits must include `Signed-off-by`:

```bash
git commit -s -m "UPSTREAM: <carry>: Add feature

Description of the change.

Signed-off-by: Your Name <your.email@example.com>"
```

---

**Last Updated:** 2025-12-11
**Maintainer:** OpenShift OLM Team
**Upstream:** https://github.com/operator-framework/operator-controller
