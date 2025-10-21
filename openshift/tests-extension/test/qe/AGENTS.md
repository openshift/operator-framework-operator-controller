# AGENTS.md

This file provides AI agents with comprehensive context about the OLM v1 QE Test Extension project to enable effective test development, debugging, and maintenance.

## Scope and Working Directory

### Applicability
This AGENTS.md applies to the **OLM v1 QE Test Cases** located at:
```
operator-framework-operator-controller/openshift/tests-extension/test/qe/
```

**IMPORTANT**: This file is specifically for the **QE migration test code** in the `test/qe/` directory, not for:
- Origin migration test code (in other directories under `tests-extension/test/`)
- Product code in the main `operator-controller` repository

### Required Working Directory
For this AGENTS.md to be effective, ensure your working directory is set to:
```bash
<repo-root>/operator-framework-operator-controller/openshift/tests-extension/test/qe/
```

### Related Directories for QE Migration

Beyond the main `test/qe/` directory, QE migration work also involves:
- `tests-extension/cmd/` - Test binary entry point and suite definitions
- `tests-extension/Makefile` - Build automation
- `tests-extension/pkg/bindata/qe/` - Embedded test data for QE tests

### Working Directory Verification for AI Agents

**Context Awareness**: This AGENTS.md may be loaded even when not actively working with QE test files (e.g., user briefly entered `test/qe/` directory and left). Apply these guidelines intelligently based on the actual task.

#### When to Apply This AGENTS.md

**ONLY apply this AGENTS.md when the user is working with QE migration test files**, identified by:
- File paths containing `tests-extension/test/qe/`
- File paths containing `tests-extension/cmd/` (suite definitions)
- File paths containing `tests-extension/pkg/bindata/qe/` (test data)
- Tasks explicitly about "OLM v1 QE tests", "QE migration", "olmv1 qe", "test extension", "olmv1-tests-ext"

**DO NOT apply this AGENTS.md when**:
- Working with files outside these directories (e.g., Origin migration tests, product code)
- User is in a different part of the repository
- Even if this AGENTS.md was previously loaded

#### Directory Check (Only for QE Test File Operations)

When the user asks to work with QE test files (files under `tests-extension/test/qe/`):

1. **Check current working directory**:
   ```bash
   pwd
   ```

2. **Verify directory alignment**:
   - Preferred: Current directory should be `tests-extension/test/qe/` or subdirectory
   - This ensures AGENTS.md context is automatically available

3. **If working directory is not aligned**:

   **Inform (don't block) the user**:
   ```
   💡 Note: Working Directory Suggestion

   You're working with QE test files under tests-extension/test/qe/,
   but your current directory is elsewhere. For better context and auto-completion:

   Consider running: cd openshift/tests-extension/test/qe/

   I can still help you, but setting the working directory correctly
   ensures I have full access to the test documentation.

   Do you want to continue in the current directory, or should I wait
   for you to switch?
   ```

**Important**: This is a suggestion, not a blocker. If the user wants to proceed, assist them normally.

### Path Structure Reference
```
operator-framework-operator-controller/                  ← OpenShift downstream product repo
└── openshift/
    └── tests-extension/                                 ← Test extension root
        ├── cmd/main.go                                  ← Test binary entry point and suite definitions
        ├── Makefile                                     ← Build automation
        ├── test/
        │   ├── qe/                                      ← THIS AGENTS.MD APPLIES HERE
        │   │   ├── AGENTS.md                            ← This file
        │   │   ├── CLAUDE.md                            ← Pointer for Claude Code
        │   │   ├── README.md                            ← Project documentation
        │   │   ├── specs/                               ← QE migration test specifications
        │   │   │   ├── olmv1_ce.go                      ← ClusterExtension tests
        │   │   │   └── olmv1_cc.go                      ← ClusterCatalog tests
        │   │   └── util/                                ← Test utilities
        │   │       ├── olmv1util/                       ← OLM v1 specific utilities
        │   │       ├── filters/                         ← Test filters
        │   │       └── ...                              ← Other utilities
        │   └── ...                                      ← Origin migration tests (NOT QE)
        ├── pkg/
        │   └── bindata/
        │       └── qe/                                  ← Embedded test data for QE tests
        └── bin/olmv1-tests-ext                          ← Compiled test binary
```

## Project Overview

This is a **Quality Engineering (QE) test extension** for OLM v1 (Operator Lifecycle Manager v1) on OpenShift. It provides end-to-end functional tests that validate OLM v1 features and functionality in real OpenShift clusters.

### Purpose
- Validate OLM v1 ClusterExtension and ClusterCatalog functionality across different OpenShift topologies
- Test operator installation, upgrade, and lifecycle management scenarios using OLM v1 APIs
- Ensure OLM v1 works correctly in various cluster configurations (SNO, standard OCP, etc.)
- Provide regression testing for OLM v1 bug fixes and enhancements

**Note**: OLM v1 currently does NOT support HyperShift and Microshift topologies. Support may be added in future releases.

### Key Characteristics
- **Framework**: Built on Ginkgo v2 BDD testing framework and OpenShift Tests Extension (OTE)
- **Test Organization**: Polarion-ID based test case management
- **Integration**: Extends `openshift-tests-extension` framework
- **API Focus**: Tests OLM v1 APIs (ClusterExtension, ClusterCatalog) rather than OLM v0 APIs

## Test Case Sources and Organization

### Two Types of Test Cases

#### 1. Migrated Cases from Origin
- **Characteristics**: All robust and stable, meeting OpenShift CI requirements
- **Contribution**: ALL contributed to openshift-tests and used in operator-controller PR presubmit jobs
- **Location**: Should NOT be implemented under `tests-extension/test/qe/specs/`
- **Note**: These cases are outside the scope of this AGENTS.md

#### 2. Migrated Cases from tests-private (QE Migration)
- **Characteristics**: Some stable, others not
- **Contribution**: Only those meeting OpenShift CI requirements can be contributed to openshift-tests
- **Location**: MUST be implemented under `tests-extension/test/qe/specs/`
- **Auto-Labeling**: Framework automatically adds `Extended` label to these cases
- **Quality Gate**: Cases not meeting CI requirements run in QE-created periodic jobs
- **Note**: This AGENTS.md focuses on these QE migration cases

### Suite Selection Logic

**For OpenShift General Jobs and PR Presubmit Jobs**:
- Select all cases by default, then exclude unwanted ones
- Migrated cases from Origin: All fit this logic
- Migrated cases from tests-private: Not all fit by default (hence the `Extended` label mechanism)
  - **IMPORTANT**: Only cases with **`Extended` AND `ReleaseGate`** labels can be used in OpenShift General Jobs and PR Presubmit Jobs
  - Cases with only `Extended` (no `ReleaseGate`) can only be used in OLM QE-defined periodic jobs

**Reference**: For OpenShift CI requirements, see [Choosing a Test Suite](https://docs.google.com/document/d/1cFZj9QdzW8hbHc3H0Nce-2xrJMtpDJrwAse9H7hLiWk/edit?tab=t.0#heading=h.tjtqedd47nnu)

## Test Suite Definitions

**IMPORTANT**: Suite definitions are sourced from **[cmd/main.go](../cmd/main.go)** and may change over time. Always refer to main.go for the most current definitions.

For detailed explanations and code examples, see **[README.md](./README.md)** section "Suite Definitions".

**Quick overview for AI agents**:

### Suites for OpenShift General Jobs and PR Presubmit Jobs
- **Suite names**: `olmv1/parallel`, `olmv1/serial`, `olmv1/slow`, `olmv1/all`
- **Selection logic**: Non-Extended OR (Extended with ReleaseGate)
- **Defined in**: [cmd/main.go](../cmd/main.go) lines 51-101

### Suites for Custom Prow Jobs (OLM QE Periodic)
```
olmv1/extended                                    # All Extended tests
├── olmv1/extended/releasegate                   # Extended + ReleaseGate
└── olmv1/extended/candidate                     # Extended without ReleaseGate
    ├── function                                 # Functional tests (excludes StressTest)
    │   ├── parallel                             # Can run concurrently
    │   ├── serial                               # Must run one at a time
    │   ├── fast                                 # Non-slow (parallel + serial)
    │   └── slow                                 # [Slow] tests
    └── stress                                   # StressTest label
```

**Key relationships**: `candidate = function + stress`, `function = parallel + serial + slow = fast + slow`

**Defined in**: [cmd/main.go](../cmd/main.go) lines 103-209, using helper functions from `test/qe/util/filters/filters.go`

## Directory Structure

```
openshift/tests-extension/
├── cmd/
│   └── main.go                   # Test binary entry point and suite definitions
│
├── test/
│   └── qe/                       # QE migration test code (THIS AGENTS.MD SCOPE)
│       ├── AGENTS.md             # This file
│       ├── CLAUDE.md             # Claude Code pointer
│       ├── README.md             # Project documentation
│       │
│       ├── specs/                # Test specifications (*.go)
│       │   ├── olmv1_ce.go       # ClusterExtension tests
│       │   └── olmv1_cc.go       # ClusterCatalog tests
│       │   └── ...               # (more test files will be added over time)
│       │
│       └── util/                 # Test utilities and helpers
│           ├── client.go         # OpenShift client wrappers
│           ├── framework.go      # Test framework setup
│           ├── tools.go          # Common test tools
│           ├── clusters.go       # Cluster detection utilities
│           ├── extensiontest.go  # Extension test helpers
│           ├── template.go       # Template processing
│           ├── architecture/     # Architecture detection
│           ├── container/        # Container client (Podman/Quay)
│           ├── filters/          # Test filters
│           │   └── filters.go    # Suite filter definitions
│           └── olmv1util/        # OLM v1 specific utilities
│               ├── catalog.go    # ClusterCatalog helpers
│               ├── helper.go     # General helpers
│               ├── icsp.go       # ImageContentSourcePolicy utilities
│               ├── networkpolicy.go # NetworkPolicy utilities
│               ├── gen_rbac.go   # RBAC generation helpers
│               └── ...           # (more utilities as needed)
│
├── pkg/
│   └── bindata/                  # Embedded test data
│       └── qe/                   # QE test bindata
│
├── bin/                          # Compiled binaries
│   └── olmv1-tests-ext           # Compiled test binary
│
└── Makefile                      # Build and test automation
```

## Test Case Migration Guide

For complete migration guidelines including code changes and label requirements, refer to **[README.md](./README.md)** section "Test Case Migration Guide".

**Quick reference for AI agents**:

### Code Changes Summary
- `exutil.By()` → `g.By()`
- `newCheck().check()` → `olmv1util.NewCheck().Check()`
- Add `exutil.` and `olmv1util.` package prefixes
- Testdata: use `"olm"` (not `"olm", "v1"`)

### Essential Labels
- `[sig-olmv1]`, `[Jira:OLM]`, `PolarionID:xxxxx` - Required in title
- `g.Label("ReleaseGate")` - For cases meeting OpenShift CI requirements (don't add to `[Disruptive]`, `[Slow]`, or `StressTest` cases)
- `[Skipped:Disconnected]`, `[Skipped:Connected]`, `[Skipped:Proxy]` - Network requirements
- `[Serial]`, `[Slow]`, `[Disruptive]` - Execution characteristics

**Note**: OLM v1 currently does NOT support Microshift and HyperShift

## Test Architecture and Patterns

### Test Structure Pattern

For complete test structure examples, refer to existing test files:
- **Standard tests**: `specs/olmv1_ce.go`, `specs/olmv1_cc.go`
- **Key patterns**: Look for `g.Describe`, `g.BeforeEach`, `g.AfterEach`, `g.It` blocks

**Basic structure**:
```go
var _ = g.Describe("[sig-olmv1][Jira:OLM] feature description", func() {
    defer g.GinkgoRecover()
    var oc = exutil.NewCLIWithoutNamespace("default")

    g.BeforeEach(func() {
      // Setup resources, skip conditions
      exutil.SkipMicroshift(oc)
      // if the user want to create project, use oc.SetupProject() here.
		exutil.SkipNoOLMv1Core(oc)
    })

    g.AfterEach(func() {
        // Cleanup resources (use defer)
    })

    g.It("PolarionID:xxxxx-test description", g.Label("ReleaseGate"), func() {
        // Test implementation
    })
})
```

**Topology-specific patterns**:
- **Microshift** (FUTURE USE - not yet supported): `exutil.SkipMicroshift(oc)`
- **HyperShift** (FUTURE USE - not yet supported): `if !exutil.IsHypershiftMgmtCluster(oc) { g.Skip(...) }`

### Skip Functions and Cluster Detection

**Note**: OLM v1 currently does NOT support Microshift and HyperShift. The related functions below are for future use when support is added.

```go
// For standard tests (skip ON Microshift) - NOT CURRENTLY NEEDED for OLM v1:
exutil.SkipMicroshift(oc)              // Skip this test on Microshift clusters

// For Microshift-specific tests (skip if NOT Microshift) - FUTURE USE:
if !exutil.IsMicroshiftCluster(oc) {
    g.Skip("it is not microshift, so skip it.")
}

// For HyperShift management cluster tests (skip if NOT HyperShift mgmt) - FUTURE USE:
if !exutil.IsHypershiftMgmtCluster(oc) {
    g.Skip("this is not a hypershift management cluster, skip test run")
}

// HyperShift-related setup (FUTURE USE when OLM v1 supports HyperShift):
exutil.EnsureHypershiftBinary(oc)      // Ensure HyperShift binary is available
exutil.ValidHypershiftAndGetGuestKubeConf(oc) // Get guest cluster kubeconfig
oc.SetGuestKubeconf(kubeconfig)        // Set guest cluster kubeconfig for test
oc.AsGuestKubeconf()                   // Use guest cluster context for operations

// AKS cluster detection:
isAKS, err := exutil.IsAKSCluster(context.TODO(), oc)

// Other skip functions:
exutil.SkipForSNOCluster(oc)           // Skip on Single Node OpenShift
exutil.IsFeaturegateEnabled(oc, "FeatureName") // Check feature gate status
```

## Local Development Workflow

For complete local development workflow, build instructions, testing procedures, PR submission requirements, and disconnected environment support, refer to **[README.md](./README.md)** section "Local Development Workflow".

**Quick reference**:
- Build: `make bindata && make build && make update-metadata`
- Find test: `./bin/olmv1-tests-ext list -o names | grep <keyword>`
- Run test: `./bin/olmv1-tests-ext run-test "<full test name>"`
- openshift-tests integration: See README.md for environment variables and suite selection
- PR requirements: See README.md for stability testing (`/payload-aggregate`) requirements

**Important for Disconnected Tests**: With IDMS/ITMS in place, tests work the same in both connected and disconnected environments. See README.md for `ValidateAccessEnvironment` usage

## Test Automation Code Requirements

For complete code quality guidelines, best practices, logging best practices, and security considerations, refer to **[README.md](./README.md)** section "Test Automation Code Requirements".

**Critical rules for AI agents**:
- ✅ Use `defer` for cleanup (BEFORE resource creation): `defer resource.Delete(oc)` then `resource.Create(oc)`
- ✅ Use case ID for resource naming (NOT random strings): `name := "test-extension-" + caseID`
- ❌ Don't use `o.Expect` inside `wait.Poll` loops (use `if err != nil { return false, err }`)
- ❌ Don't execute logic in `g.Describe` blocks (only initialization, move logic to `g.BeforeEach`)
- ❌ Don't use quotes in test titles (breaks XML parsing)
- ❌ Don't put large log outputs in error messages (use proper log messages instead of `o.Expect` with large output)

## Key Utilities

For complete utility APIs and usage examples, refer to the source code and existing tests:

### `exutil` Package
**Location**: `util/` directory (e.g., `util/client.go`, `util/framework.go`, `util/tools.go`, `util/clusters.go`)

**Key functions**:
- CLI management: `NewCLI()`, `KubeConfigPath()`
- Resource operations: `OcAction()`, `OcCreate()`, `OcDelete()`, `PatchResource()`
- Cluster detection: `IsSNOCluster()`, `IsROSA()`, `IsTechPreviewNoUpgrade()`, `IsFeaturegateEnabled()`
- Skip functions: `SkipMicroshift()` (FUTURE USE), `SkipForSNOCluster()`

### `olmv1util` Package
**Location**: `util/olmv1util/` directory (e.g., `util/olmv1util/catalog.go`, `util/olmv1util/helper.go`)

**Key types and methods**:
- `ClusterCatalogDescription`: Create, Delete, WaitCatalogStatus
- `ClusterExtensionDescription`: Create, Delete, WaitClusterExtensionCondition
- `NewCheck()`: Validation helper for ClusterExtension/ClusterCatalog state

**Usage examples**: See existing test files in `specs/olmv1_ce.go` and `specs/olmv1_cc.go`

## Anti-Patterns to Avoid

For complete anti-patterns with detailed code examples and explanations, refer to **[README.md](./README.md)** section "Test Automation Code Requirements".

**Common mistakes for AI agents to avoid**:
- ❌ No cleanup: Always use `defer resource.Delete(oc)` BEFORE `resource.Create(oc)`
- ❌ Hardcoded names: Use case ID for naming: `name := "test-extension-" + caseID`
- ❌ Missing timeouts: Always specify timeout for Wait functions
- ❌ Hard sleeps: Use Wait functions instead of `time.Sleep()`
- ❌ `o.Expect` in `wait.Poll`: Use `if err != nil { return false, err }` pattern instead

**See existing test patterns**: `specs/olmv1_ce.go` and `specs/olmv1_cc.go`

## Quick Reference

For complete workflow including openshift-tests integration and PR requirements, see **[README.md](./README.md)** section "Local Development Workflow".

### Build and Run
```bash
make bindata && make verify && make build && make update-metadata    # Full build

./bin/olmv1-tests-ext list -o names | grep "keyword"  # Find test
./bin/olmv1-tests-ext run-test "<full test name>"     # Run test
```

### Test Naming Convention
```
[sig-olmv1][Jira:OLM] OLMv1 <feature> PolarionID:XXXXX-[Skipped:XXX]description[Serial|Slow|Disruptive]
```

### Key Labels (See README.md for complete list)
- `ReleaseGate` - Promotes Extended case to openshift-tests (don't add to `[Disruptive]`, `[Slow]`, or `StressTest`)
- `Extended` - Auto-added to cases under test/qe/specs/
- `StressTest` - Stress testing
- `NonHyperShiftHOST` - Skip on HyperShift hosted clusters (FUTURE USE)

### Key OLM v1 Resources
- **ClusterCatalog**: Cluster-scoped catalog of operator bundles
- **ClusterExtension**: Cluster-scoped operator installation and management

### Key OLM v1 Namespaces (OpenShift)
- **openshift-operator-controller**: Operator controller components
- **openshift-catalogd**: Catalogd components

## Resources

- [OLM v1 OpenShift Product Code](https://github.com/openshift/operator-framework-operator-controller)
- [Ginkgo v2 Documentation](https://onsi.github.io/ginkgo/)
- [OpenShift Tests Extension](https://github.com/openshift-eng/openshift-tests-extension)
- [Test Extensions in Origin](https://github.com/openshift/origin/blob/main/docs/test_extensions.md)
- [OpenShift CI Requirements](https://docs.google.com/document/d/1cFZj9QdzW8hbHc3H0Nce-2xrJMtpDJrwAse9H7hLiWk/edit?tab=t.0#heading=h.tjtqedd47nnu)

## Debugging

**Investigation Priority** when tests fail:
1. **First**: Check test code in `tests-extension/test/qe/`
2. **Second**: Check test utilities in `tests-extension/test/qe/util/olmv1util/`
3. **Third**: Check resource status and conditions via `oc describe`
4. **Fourth**: Check operator-controller and catalogd logs
5. **Last**: Refer to product code to understand expected behavior

**For deeper investigation** (when you need to refer to product code):
1. **Locate product code**: See **Product Code References** section below
2. **Trace code flow**: Use product code to understand expected behavior
3. **Compare implementation**: Check if test expectations match product implementation
4. **Check recent changes**: Look for recent commits that might have changed behavior

**Key Namespaces** (OpenShift):
- `openshift-operator-controller`: operator-controller components
- `openshift-catalogd`: catalogd components

**Common Debugging Commands**:
```bash
# Check resource status
oc get clusterextension
oc get clustercatalog
oc describe clusterextension <name>
oc describe clustercatalog <name>

# Check logs
oc logs -n openshift-operator-controller deployment/operator-controller-controller-manager -f
oc logs -n openshift-catalogd deployment/catalogd-controller-manager -f
```

## Notes for AI Agents

### Suggesting Test Locations

When discussing whether a feature needs testing:

**✅ DO**: Provide simple, focused guidance on QE test placement
- Example: "If you need to write QE tests for this functionality, they should go in `tests-extension/test/qe/specs/`."
- Keep suggestions within the scope of this AGENTS.md (QE tests only)

**❌ DON'T**:
- Discuss DEV test locations (e.g., unit tests in product code directories)
- Explain the difference between QE and DEV tests unless explicitly asked
- Provide detailed test categorization unless the user is actively writing tests

**Remember**: This AGENTS.md is for QE test code in `tests-extension/test/qe/` only. Product code testing (DEV tests) is outside this scope.

### Critical Points

1. **Test Scope**:
   - This AGENTS.md applies ONLY to QE migration test code under `test/qe/`
   - Origin migration tests (outside `test/qe/`) have different patterns and are NOT covered here

2. **Suite Definitions Source**:
   - Always check `cmd/main.go` for current suite definitions
   - Suite qualifiers may change over time

3. **Extended Label Mechanism**:
   - Tests under `test/qe/specs/` automatically get `Extended` label
   - Only `Extended + ReleaseGate` cases can be used in OpenShift General Jobs
   - Extended cases without `ReleaseGate` run only in QE periodic jobs

4. **ReleaseGate is Critical**:
   - Determines if Extended case can be used in OpenShift General Jobs and PR Presubmit Jobs
   - All cases are executed via `openshift-tests` command

5. **Most Failures are Test Code Issues**:
   - Always investigate test code first before looking at product code
   - Refer to Debugging section for investigation priority

### Test Development Guidelines

1. **Component Tag**: Always use `[sig-olmv1]` (not `[sig-operator]`)
2. **Utilities**: Use `olmv1util` package (not `olmv0util`)
3. **API Focus**: Test OLM v1 APIs (ClusterExtension, ClusterCatalog) not OLM v0 APIs
4. **Cleanup**: Always use defer for cleanup to ensure resources are removed
5. **Resource Naming**: Use Polarion case ID for naming resources (NOT random strings)
   - Extract case ID from test title: `PolarionID:12345` → `caseID := "12345"`
   - Apply to all resources: `namespace := "test-ns-" + caseID`
   - Benefits: Easier debugging, consistent naming, traceable to test cases
6. **Suite Logic**: Understand the qualifier logic for different test suites
   - Refer to Test Suite Definitions section for suite hierarchy
   - Understand which suite your test belongs to based on labels
7. **Feature Gates**: For tests depending on feature gates, see **[README.md](./README.md)** section "Label Requirements" for detailed handling patterns:
   - Case 1: Test only runs when feature gate is enabled → Add `[OCPFeatureGate:xxxx]` in title
   - Case 2: Test runs with/without gate but different behaviors → Use `IsFeaturegateEnabled` check (no label)
   - Case 3: Test runs same way regardless of gate → No label, no check

### Cluster Topologies

**Note**: OLM v1 currently supports only a subset of OpenShift topologies. Support for additional topologies may be added in future releases.

**Currently Supported**:
- **Standard OCP**: Regular OpenShift clusters
- **SNO (Single Node OpenShift)**: Single-node clusters

**NOT Currently Supported** (for future releases):
- **Microshift**: Lightweight OpenShift for edge (not yet supported by OLM v1)
- **HyperShift Hosted**: Hosted control plane clusters (not yet supported by OLM v1)
- **HyperShift Management**: Management clusters for hosted control planes (not yet supported by OLM v1)

**Network Connectivity**:
- **Connected**: Full internet access
- **Disconnected**: No internet access (air-gapped)
- **Proxy**: Internet access through proxy

Use skip labels in test titles for topology-specific tests:
- `[Skipped:Disconnected]`: Test requires internet access
- `[Skipped:Connected]`: Test requires disconnected environment
- `[Skipped:Proxy]`: Test incompatible with proxy

### Common Pitfalls

**Test Code Issues**:
1. ❌ **Don't** use `o.Expect` inside `wait.Poll` loops (causes panic)
2. ❌ **Don't** use quotes in test titles (breaks XML parsing)
3. ❌ **Don't** execute logic in `g.Describe` blocks (only initialization)
4. ❌ **Don't** forget to add `Extended` label (but it's automatic for `test/qe/specs/`)
5. ❌ **Don't** add `ReleaseGate` to `[Disruptive]`, `[Slow]`, or `StressTest` cases
6. ❌ **Don't** forget cleanup in `g.AfterEach` with defer
7. ❌ **Don't** assume namespace exists - create with unique name
8. ❌ **Don't** hardcode resource names - use case ID for naming

**OLM v1 Specific Issues**:
9. ❌ **Don't** assume multi-tenancy - OLM v1 explicitly does NOT support it
10. ❌ **Don't** forget to wait for ClusterCatalog `Serving` status before creating ClusterExtension
11. ❌ **Don't** assume catalog is immediately available - allow time for unpacking
12. ❌ **Don't** ignore resolution errors - check ClusterExtension status conditions

### Best Practices

**General Test Practices**:
1. ✅ **Do** check suite definitions in `cmd/main.go` before adding tests
2. ✅ **Do** use case ID for naming resources (NOT random strings)
3. ✅ **Do** add proper PolarionID to all test cases
4. ✅ **Do** use skip functions for topology-specific tests
5. ✅ **Do** register defer cleanup BEFORE creating resources
   - Pattern: `defer resource.Delete(oc)` then `resource.Create(oc)`
   - Why: Ensures cleanup even if Create partially succeeds then fails
6. ✅ **Do** test locally with `olmv1-tests-ext` before submitting PR
7. ✅ **Do** test with `openshift-tests` to verify suite selection
8. ✅ **Do** run stability tests (`/payload-aggregate`) for ReleaseGate cases
9. ✅ **Do** update metadata after test name changes

**OLM v1 Specific Practices**:
9. ✅ **Do** wait for ClusterCatalog to reach `Serving` status
10. ✅ **Do** check ClusterExtension status conditions for debugging
11. ✅ **Do** verify bundle resources are created in target namespace
12. ✅ **Do** use meaningful names that trace back to test case ID
13. ✅ **Do** clean up ClusterExtension and ClusterCatalog in defer blocks
14. ✅ **Do** understand the data flow: Catalog → Resolution → Installation

### Build and Run

For complete workflow and detailed commands, refer to **[README.md](./README.md)** section "Local Development Workflow" and the **[Quick Reference](#quick-reference)** section above.

**Essential pattern for AI agents**:
1. Build: `make bindata && make build && make update-metadata`
2. Find test: `./bin/olmv1-tests-ext list -o names | grep <keyword>`
3. Run locally: `./bin/olmv1-tests-ext run-test "<full test name>"`
4. Test with openshift-tests: See README.md for environment variables and suite selection
5. Run stability tests: `/payload-aggregate` for ReleaseGate cases (see README.md for details)

### Working Directory Context

**Remember**: This AGENTS.md is specifically for QE migration tests under `test/qe/`. If the user is working with:
- Origin migration tests (outside `test/qe/`) → This AGENTS.md does NOT apply
- Product code → This AGENTS.md does NOT apply
- Build infrastructure → This AGENTS.md may partially apply (for suite definitions in `cmd/main.go`)
