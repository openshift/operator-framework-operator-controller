# AGENTS.md - Guide for AI Tools

This document provides critical guidance for AI agents working with the OpenShift operator-controller fork.

---

## Critical: This is a Downstream Fork

This repository (`openshift/operator-framework-operator-controller`) is OpenShift's downstream fork of `operator-framework/operator-controller`.

**Upstream Repository:** https://github.com/operator-framework/operator-controller

**Key Constraints for AI Agents:**
- This fork is **synced daily** against upstream by automated tooling
- Changes must use **strict commit message conventions** enforced by CI
- **Minimize divergence** from upstream - every downstream change creates maintenance burden
- **DO NOT create `UPSTREAM: <PR>:` backport commits** - automated tooling handles upstream sync

**Rebase Tooling:** https://github.com/openshift/operator-framework-tooling

---

## Commit Message Conventions (MANDATORY)

**ALL commits MUST use one of these prefixes. CI checks will FAIL non-compliant PRs.**

### `UPSTREAM: <carry>:`

**Use for:** OpenShift-specific changes to non-generated content that must persist across rebases.

**Important:** 99% of carry changes should be in the `openshift/` directory. Code outside `openshift/` should generally be modified upstream and synced downstream.

**Examples:**
```
UPSTREAM: <carry>: Add OpenShift-specific test for disconnected environments
UPSTREAM: <carry>: Update OCP catalogs to v4.21
UPSTREAM: <carry>: Use busybox/httpd to simulate probes
```

**Note:** Creates ongoing maintenance burden. Only use when change cannot go upstream.

### `UPSTREAM: <drop>:`

**Use for:** Temporary commits that will be dropped on the next upstream sync.

**When to use:**
- Pulling down an upstream fix early before the next scheduled sync
- Resolving merge conflicts that cannot be fixed by local regeneration
- Changes that are easier to recreate than rebase/cherry-pick (e.g., removing upstream configuration)

**Examples:**
```
UPSTREAM: <drop>: make manifests
UPSTREAM: <drop>: go mod tidy
UPSTREAM: <drop>: Remove upstream GitHub configuration
```

**Note:** These commits are dropped during the next sync, so they're temporary by design.

### `UPSTREAM: <PR>:`

**AI AGENTS: DO NOT CREATE THESE COMMITS.**

This prefix is **ALMOST NEVER USED** on the main branch because it creates conflicts on the next upstream sync. The bumper tool automatically synchronizes upstream changes during scheduled rebases.

**For urgent fixes:** Use `UPSTREAM: <drop>:` instead to pull the fix early - it will be dropped on the next sync when the real upstream commit comes down.

**Note:** Cherry-picking with this prefix may be done on release branches, but since most work is on main, this is extremely uncommon.

---

## Decision Tree for Changes

| Change Type | Action | Commit Prefix | AI Agent Action |
|------------|--------|--------------|----------------|
| General bug fix | Fix upstream, then let tooling synchronize | None | **DO NOT create PR** - direct user to upstream |
| OpenShift-specific test | Downstream only | `UPSTREAM: <carry>:` | Create in `openshift/tests-extension/` |
| OpenShift-specific config | Downstream only | `UPSTREAM: <carry>:` | Create in `openshift/` |
| Generated files (to fix failures) | Only when necessary | `UPSTREAM: <drop>:` | **Only if absolutely necessary** |

---

## Repository Structure

### Upstream Code (Minimize Changes)
- `api/`, `cmd/`, `internal/`, `pkg/`, `test/`, `helm/`, `docs/`
- **Before modifying:** Can this be fixed upstream? Can this go in `openshift/`?

### Downstream Code (Safe to Modify)
- `openshift/` - ALL OpenShift-specific code and configuration
  - `openshift/Makefile` - Downstream build targets
  - `openshift/*.Dockerfile` - OpenShift container builds
  - `openshift/tests-extension/` - OpenShift Test Extension (OTE) tests
  - `openshift/dev_notes.md` - Downstreaming process documentation
  - `openshift/AGENTS.md` - This file

---

## Rules for AI Agents

### DO:
1. **Always use correct `UPSTREAM:` prefix** - CI will fail without it
2. **Prefer `openshift/` directory** for new OpenShift-specific code
3. **Check upstream first** before fixing bugs - should it go upstream?
4. **Separate commits:** Never mix code changes with generated files
5. **Run verification:** `make verify` and `make test-unit` before completion

### DO NOT:
1. **DO NOT create `UPSTREAM: <PR>:` backport commits** - tooling handles this
2. **DO NOT use `UPSTREAM: <drop>:` routinely** - only for critical failures
3. **DO NOT modify upstream code** without confirming it's OpenShift-specific
4. **DO NOT hand-edit generated files** - use `make manifests` instead
5. **DO NOT mix code and generated changes** in same commit
6. **DO NOT skip commit prefixes** - CI will reject the PR

---

## Generated Files (Special Handling)

**These files are generated - DO NOT hand-edit:**
- CRD manifests in `helm/olmv1/base/*/crd/`
- Manifests in `manifests/`
- `api/v1/zz_generated.deepcopy.go`
- CRD docs in `docs/api-reference/`

**Workflow for AI Agents:**
1. Make code changes
2. Run `make manifests` locally
3. Verify with `make verify` and `make test-unit`
4. **Commit ONLY the code changes** with `UPSTREAM: <carry>:`
5. **DO NOT commit generated files** unless absolutely necessary to resolve CI failures or merge conflicts

**Only commit regenerated files with `UPSTREAM: <drop>:` when:**
- CI is failing and requires the generated files
- Merge conflicts cannot be resolved by local regeneration
- Multiple PRs have created unmergeable conflicts in generated content

---

## Build and Test Commands

### Verification
```bash
# Root directory
make build
make test-unit
make verify

# OpenShift directory
cd openshift
make verify
make test-e2e
```

### Regenerating Files Locally
```bash
# Always run these after making changes
make manifests
make generate

# Verify tests still pass
make verify
make test-unit
```

**Important:** Run these locally but do NOT commit the generated files unless absolutely necessary to resolve failures.

### Dependencies
```bash
# Run locally to keep dependencies in sync
go mod tidy

# DO NOT commit unless absolutely necessary to resolve failures
```

---

## Common Tasks

### Adding OpenShift-Specific Tests

Add tests to `openshift/tests-extension/`:

```bash
# Create test file in openshift/tests-extension/
# Commit:
git commit -s -m "UPSTREAM: <carry>: Add OTE test for disconnected installation

This test validates OLMv1 behavior in OpenShift disconnected environments.

Signed-off-by: Your Name <your@example.com>"

# Verify:
cd openshift
make test-e2e
```

### Updating OpenShift Configurations

```bash
# Make changes to openshift/ directory files
git commit -s -m "UPSTREAM: <carry>: Update OCP catalogs to v4.22

Updates catalog references to OpenShift 4.22 release catalogs.

Signed-off-by: Your Name <your@example.com>"
```

---

## Examples: Good vs. Bad

### ❌ BAD: Creating Backport Commit

```bash
git commit -m "UPSTREAM: 567: Fix reconciliation loop"
```

**Problem:** AI agents should NOT create backport commits. Tooling handles this.

### ✅ GOOD: Directing to Upstream

**User:** "Fix the reconciliation bug"

**AI:** "This appears to be a general operator-controller issue. I recommend:
1. Check if fixed in upstream `operator-framework/operator-controller`
2. If not, open upstream PR first
3. Automated tooling will sync it downstream on next rebase

Is this OpenShift-specific? If yes, I can create a carry patch."

---

### ❌ BAD: Mixing Code and Generated Files

```
UPSTREAM: <carry>: Fix auth and update manifests

- Modified internal/auth/openshift.go
- Regenerated CRD manifests
```

**Problem:** Mixes code changes with generated files.

### ✅ GOOD: Code Change Only

```
UPSTREAM: <carry>: Fix OpenShift authentication integration

Modified internal/auth/openshift.go

Signed-off-by: Your Name <your@example.com>
```

**After committing:** Run `make manifests` locally to regenerate files. Do NOT commit generated files unless absolutely necessary to resolve failures.

---

### ❌ BAD: Routine Use of `UPSTREAM: <drop>`

```
UPSTREAM: <drop>: make manifests
```

**Problem:** Only use when absolutely necessary to resolve failures, not routinely after every change.

### ✅ GOOD: Only When Necessary

Use `UPSTREAM: <drop>:` for generated files **only** when required to resolve merge conflicts or critical failures that cannot be fixed by local regeneration.

---

## Enforcement

**CI checks by commitchecker will FAIL PRs that do not follow commit conventions.**

Non-compliant PRs will be rejected by maintainers.

---

## Resources

- **Upstream AGENTS.md:** Root of this repository (general operator-controller info)
- **Downstreaming tooling:** https://github.com/openshift/operator-framework-tooling
- **Downstreaming guide:** `openshift/dev_notes.md`
- **Origin tests guide:** `openshift/HOWTO-origin-tests.md`

---

## Quick Reference

```bash
# Verify changes
cd openshift
make verify
make test-e2e

# Regenerate files
make manifests
make generate

# Check commit history
git log --oneline --grep="UPSTREAM: <carry>"

# Proper commit format
git commit -s -m "UPSTREAM: <carry>: Description

Detailed explanation.

Signed-off-by: Your Name <your@example.com>"
```

---

## Summary

**This is a downstream fork that gets regularly rebased against upstream.**

Before suggesting any change, AI agents MUST understand:

1. **Is this upstream or OpenShift-specific?** If upstream, direct to upstream repo
2. **What commit prefix is required?** Use `UPSTREAM: <carry>:` for OpenShift-specific changes
3. **DO NOT create `UPSTREAM: <PR>:` backports** - tooling handles upstream sync
4. **DO NOT use `UPSTREAM: <drop>:` routinely** - only for critical failures
5. **Minimize divergence** - every carry patch creates maintenance burden

**When in doubt:** Ask if the change is OpenShift-specific or should go upstream first.

---

**Last Updated:** 2025-12-11
**Maintainer:** OpenShift OLM Team
**Upstream:** https://github.com/operator-framework/operator-controller
