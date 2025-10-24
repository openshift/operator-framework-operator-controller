## Overview

When creating test cases based on OTE (OpenShift Tests Extension) in operator-controller, there are two sources:

### 1. Migrated Cases from Origin
- These cases are all robust and stable, meeting OpenShift CI requirements
- All of them are contributed to openshift-tests and used in operator-controller PR presubmit jobs

### 2. Migrated Cases from tests-private
- Some of these cases are stable, others are not
- Only those meeting OpenShift CI requirements can be contributed to openshift-tests and used in operator-controller PR presubmit jobs
- The remaining cases that don't meet OpenShift CI requirements are run by QE-created periodic jobs

## Suite Selection Logic

Test suites for openshift-tests and PR presubmit jobs select all cases by default, then exclude unwanted ones:
- **migrated cases from Origin**: All can be contributed to openshift-tests (fits this logic)
- **migrated cases from tests-private**: Not all can be contributed to openshift-tests by default (doesn't fit this logic)

We need to identify all cases from tests-private among all cases, then mark which cases can be contributed to openshift-tests and PR presubmit jobs.

> **Note**: For OpenShift CI requirements, refer to: [Choosing a Test Suite](https://docs.google.com/document/d/1cFZj9QdzW8hbHc3H0Nce-2xrJMtpDJrwAse9H7hLiWk/edit?tab=t.0#heading=h.tjtqedd47nnu)

## Implementation Strategy

### Test Case Organization

1. **Case from tests-private Identification**: These cases must be implemented under `openshift/tests-extension/test/qe/specs/`
   - Test framework automatically adds `Extended` label to the cases
   - Enables automatic cases identification without requiring authors to add labels

2. **Case from Origin Placement**: These cases should NOT be implemented under `openshift/tests-extension/test/qe/specs/`

3. **OpenShift CI Compatibility for cases from tests-private**: If the author believes a case meets OpenShift CI requirements, add the `ReleaseGate` label:
   ```go
   g.It("xxxxxx", g.Label("ReleaseGate"), func() {
   ```
   - This makes the case equivalent to origin cases for openshift-tests
   - Test framework automatically sets these cases without `ReleaseGate` as `Informing`
   - For the cases with `ReleaseGate` that need `Informing`, add:
     ```go
     import oteg "github.com/openshift-eng/openshift-tests-extension/pkg/ginkgo"
     g.It("xxxxxx", g.Label("ReleaseGate"), oteg.Informing(), func() {
     ```

## Suite Definitions

### Suites for openshift-tests and PR presubmit jobs:

#### Parallel Suite
```go
	ext.AddSuite(e.Suite{
		Name:    "olmv1/parallel",
		Parents: []string{"openshift/conformance/parallel"},
		Qualifiers: []string{
			`((!labels.exists(l, l=="Extended")) || (labels.exists(l, l=="Extended") && labels.exists(l, l=="ReleaseGate"))) &&
			!(name.contains("[Serial]") || name.contains("[Slow]"))`,
		},
	})
```

#### Serial Suite
```go
	ext.AddSuite(e.Suite{
		Name:    "olmv1/serial",
		Parents: []string{"openshift/conformance/serial"},
		Qualifiers: []string{
			`((!labels.exists(l, l=="Extended")) || (labels.exists(l, l=="Extended") && labels.exists(l, l=="ReleaseGate"))) &&
			(name.contains("[Serial]") && !name.contains("[Disruptive]") && !name.contains("[Slow]"))`,
			// refer to https://github.com/openshift/origin/blob/main/pkg/testsuites/standard_suites.go#L456
		},
	})
```

#### Slow Suite
```go
	ext.AddSuite(e.Suite{
		Name:    "olmv1/slow",
		Parents: []string{"openshift/optional/slow"},
		Qualifiers: []string{
			`((!labels.exists(l, l=="Extended")) || (labels.exists(l, l=="Extended") && labels.exists(l, l=="ReleaseGate"))) &&
			name.contains("[Slow]")`,
		},
	})
```

#### All Suite
```go
	ext.AddSuite(e.Suite{
		Name: "olmv1/all",
		Qualifiers: []string{
			`(!labels.exists(l, l=="Extended")) || (labels.exists(l, l=="Extended") && labels.exists(l, l=="ReleaseGate"))`,
		},
	})
```

### Suites for Custom Prow jobs:

#### Extended All Suite
```go
	ext.AddSuite(e.Suite{
		Name: "olmv1/extended",
		Qualifiers: []string{
			`labels.exists(l, l=="Extended")`,
		},
	})
```

#### Extended ReleaseGate Suite
```go
	ext.AddSuite(e.Suite{
		Name: "olmv1/extended/releasegate",
		Qualifiers: []string{
			`labels.exists(l, l=="Extended") && labels.exists(l, l=="ReleaseGate")`,
		},
	})
```

#### Extended Candidate Suite
```go
	ext.AddSuite(e.Suite{
		Name: "olmv1/extended/candidate",
		Qualifiers: []string{
			`labels.exists(l, l=="Extended") && !labels.exists(l, l=="ReleaseGate")`,
		},
	})
```

#### Extended Candidate Parallel Suite
```go
	ext.AddSuite(e.Suite{
		Name: "olmv1/extended/candidate/parallel",
		Qualifiers: []string{
			`(labels.exists(l, l=="Extended") && !labels.exists(l, l=="ReleaseGate") && !labels.exists(l, l=="StressTest")) &&
			!(name.contains("[Serial]") || name.contains("[Slow]"))`,
		},
	})
```

#### CExtended Candidate Serial Suite
```go
	ext.AddSuite(e.Suite{
		Name: "olmv1/extended/candidate/serial",
		Qualifiers: []string{
			`(labels.exists(l, l=="Extended") && !labels.exists(l, l=="ReleaseGate") && !labels.exists(l, l=="StressTest")) &&
			(name.contains("[Serial]") && !name.contains("[Slow]"))`,
		},
	})
```

#### Extended Candidate Slow Suite
```go
	ext.AddSuite(e.Suite{
		Name: "olmv1/extended/candidate/slow",
		Qualifiers: []string{
			`(labels.exists(l, l=="Extended") && !labels.exists(l, l=="ReleaseGate") && !labels.exists(l, l=="StressTest")) &&
			name.contains("[Slow]")`,
		},
	})
```

## Test Case Migration Guide

### A. Code Changes for Migrated Cases

All migrated test case code needs the following changes to run in the new test framework:

1. Change `exutil.By()` to `g.By()`
2. Change `newCheck()` to `olmv1util.NewCheck()`, change `check(oc)` to `Check(oc)`
3. Change `getResource()` to `olmv1util.Get()`
4. Change `patchResource()` to `exutil.PatchResource()`
5. Change `getRandomString()` to `exutil.GetRandomString()`
6. Change `asAdmin`, `withoutNamespace`, `contain`, `ok` to `exutil.AsAdmin`, `exutil.WithoutNamespace`, `exutil.Contain`, `exutil.Ok`
7. Change testdata from `"olm", "v1"` to `"olm"`

### B. Label Requirements for Migrated and New Cases

#### Required Labels
1. **Component annotation**: Add `[sig-olmv1]` in case title
2. **Jira Component**: Add `[Jira:OLM]` in case title
3. **OpenShift CI compatibility**: If you believe the case meets OpenShift CI requirements, add `ReleaseGate` label to Ginkgo
   - **Note**: Don't add `ReleaseGate` if case title contains `Disruptive` or `Slow`, or labels contain `StressTest`

#### Optional Label for Migration and New
4. **LEVEL0**: Use Ginkgo label `g.Label("LEVEL0")`
5. **Author**: Deprecated
6. **ConnectedOnly**: Add `[Skipped:Disconnected]` in title
7. **DisconnectedOnly**: Add `[Skipped:Connected][Skipped:Proxy]` in title
8. **Case ID**: change to `PolarionID:xxxxxx`
9. **Importance**: Deprecated
10. **NonPrerelease**: Deprecated
    - **Longduration**: Change to `[Slow]` in case title
    - **ChkUpg**: Not supported (openshift-tests upgrade differs from OpenShift QE)
11. **VMonly**: Deprecated
12. **Slow, Serial, Disruptive**: Preserved
13. **DEPRECATED**: Deprecated, corresponding cases deprecated. Use `IgnoreObsoleteTests` for deprecation after addition
14. **CPaasrunOnly, CPaasrunBoth, StagerunOnly, StagerunBoth, ProdrunOnly, ProdrunBoth**: Deprecated
15. **StressTest**: Use Ginkgo label `g.Label("StressTest")`
16. **NonHyperShiftHOST**: Use Ginkgo label `g.Label("NonHyperShiftHOST")` or use `IsHypershiftHostedCluster` judgment, then skip
17. **HyperShiftMGMT**: Deprecated. For cases needing hypershift mgmt execution, use `g.Label("NonHyperShiftHOST")` and `ValidHypershiftAndGetGuestKubeConf` validation (to be provided when OLMv1 supports hypershift)
18. **MicroShiftOnly**: Deprecated. For cases not supporting microshift, use `SkipMicroshift` judgment, then skip
19. **ROSA**: Deprecated. Three ROSA job types:
    - `rosa-sts-ovn`: equivalent to OCP
    - `rosa-sts-hypershift-ovn`: equivalent to hypershift hosted
    - `rosa-classic-sts`: doesn't use openshift-tests
20. **ARO**: Deprecated. All ARO jobs based on HCP are equivalent to hypershift hosted (don't actually use openshift-test)
21. **OSD_CCS**: Deprecated. Only one job type: `osd-ccs-gcp` equivalent to OCP
22. **Feature Gates**: Handle test cases based on their feature gate requirements:

    **Case 1: Test only runs when feature gate is enabled**
    - The test should not execute if the feature gate is disabled
    - Add `[OCPFeatureGate:xxxx]` in `g.It` title (where xxxx is feature gate name)
    - Or use `IsFeaturegateEnabled` check, then skip if disabled
    - Remove label/check when feature no longer requires gate
    
    **Case 2: Test runs with/without feature gate but with different behaviors**
    - The test executes regardless of feature gate status, but behaves differently
    - Use `IsFeaturegateEnabled` check to handle different behaviors
    - Do NOT add `[OCPFeatureGate:xxxx]` label
    - Remove `IsFeaturegateEnabled` check when feature no longer requires gate
    
    **Case 3: Test runs with/without feature gate with same behavior**
    - The test executes the same way regardless of feature gate status
    - Do NOT use `IsFeaturegateEnabled` check
    - Do NOT add `[OCPFeatureGate:xxxx]` label

## Test Automation Code Requirements

Consider these requirements when writing and reviewing code:

### Security Considerations
- Does the test case generate sensitive information in logs?
- Does the code contain sensitive information in output or commands?

### Test Isolation
- Will this test case affect other test executions?
- Will this test case be affected by other test executions?

### Labeling and Cleanup
- Are correct labels applied?
- What changes does this case make to the cluster?
- Can changes be restored for both normal and abnormal exits?
- During recovery, are both actions and results correct?
- Should recovery restore to predetermined or dynamically determined values?

### Logging Best Practices
- Avoid excessive logs or large error messages
- Don't put large log outputs in error messages(use proper log messages instead). Don't use `o.Expect` to assert large messages (appears in error message on failure)
- Avoid logging `oc logs` output directly

### Code Quality
- Don't modify shared libraries (e.g., Ginkgo) or global settings affecting other tests
- Don't execute logic code in `g.Describe` except for initing oc, and move to `g.BeforeEach`
- Don't use single/double quotes in case titles (causes XML parse failures)
- Avoid `o.Expect` in `wait.Poll`:
  ```go
  // Wrong:
  wait.PollUntilContextTimeout(context.TODO(), time.Second, time.Minute, false, func(ctx context.Context) (bool, error) {
		response, err := c.AuthorizationV1().SelfSubjectAccessReviews().Create(context.Background(), review, metav1.CreateOptions{})
		o.Expect(err).NotTo(o.HaveOccurred()) // in wait.Poll
		return response.Status.Allowed == allowed, nil
	})
  
  // Correct:
  wait.PollUntilContextTimeout(context.TODO(), time.Second, time.Minute, false, func(ctx context.Context) (bool, error) {
		response, err := c.AuthorizationV1().SelfSubjectAccessReviews().Create(context.Background(), review, metav1.CreateOptions{})
		if err != nil {
			return false, err
		}
		return response.Status.Allowed == allowed, nil
	})
  ```

## Local Development Workflow

### Before Submitting PR

1. **Build and compile**:
   ```bash
   make bindata
   make build
   ```

2. **Check test name**:
   ```bash
   # List all test names and search for your test using a keyword
   ./bin/olmv1-tests-ext list -o names | grep "keyword_from_your_test_name"
   
   # Example: If your test is about "catalog installation", search for:
   ./bin/olmv1-tests-ext list -o names | grep "catalog"
   # This will show the full test name like:
   # [sig-olmv1][Jira:OLM] OLMv1 catalog installation should succeed
   ```

3. **Run test locally**:
   ```bash
   ./bin/olmv1-tests-ext run-test <full test name>
   ```

4. **Test with openshift-tests**:
   - Switch to origin repo
   - Follow [test extensions documentation](https://github.com/openshift/origin/blob/main/docs/test_extensions.md)
   - Set environment variables:
     ```bash
     OPENSHIFT_TESTS_DISABLE_CACHE=1
     EXTENSION_BINARY_OVERRIDE_INCLUDE_TAGS=tests,olm-operator-controller
     EXTENSION_BINARY_OVERRIDE_OLM_OPERATOR_CONTROLLER=<path to repo>/operator-framework-operator-controller/openshift/tests-extension/bin/olmv1-tests-ext
     EXTENSIONS_PAYLOAD_OVERRIDE=<ocp arm payload> # For AMD cluster with ARM laptop:
     # EXTENSIONS_PAYLOAD_OVERRIDE=registry.ci.openshift.org/ocp-arm64/release-arm64:4.20.0-0.nightly-arm64-2025-08-31-123924
     ```
   - Run appropriate suite based on your test characteristics:
     ```bash
     # Choose the suite that matches your test type:
     
     # For parallel tests (most common):
     ./openshift-tests run olmv1/parallel --monitor watch-namespaces
     
     # For parallel tests which does not contributed to openshift-tests
     ./openshift-tests run olmv1/extended/candidate/parallel --monitor watch-namespaces
     ```

5. **Update metadata**:
   ```bash
   make update-metadata
   ```
   - If test name changed, refer to "How to Keep Test Names Unique"

6. **Create PR**

### PR Submission Requirements

#### Pre-submission Checks
1. Check failed presubmit jobs - verify both your new cases and whether other case failures are caused by your changes

#### Stability Testing
2. **For parallel cases** contributing to openshift-tests:
   ```bash
   /payload-aggregate periodic-ci-openshift-release-master-ci-<release version>-e2e-gcp-ovn-techpreview 5
   # Example: /payload-aggregate periodic-ci-openshift-release-master-ci-4.20-e2e-gcp-ovn-techpreview 5
   
   /payload-aggregate periodic-ci-openshift-release-master-ci-<release version>-e2e-gcp-ovn 5
   ```

3. **For serial cases** contributing to openshift-tests:
   ```bash
   /payload-aggregate periodic-ci-openshift-release-master-ci-<release version>-e2e-gcp-ovn-techpreview-serial 5
   # Example: /payload-aggregate periodic-ci-openshift-release-master-ci-4.20-e2e-gcp-ovn-techpreview-serial 5
   
   /payload-aggregate periodic-ci-openshift-release-master-ci-<release version>-e2e-gcp-ovn-serial 5
   ```

## How to Keep Test Names Unique

OTE requires unique test names. If you want to modify a test name after merging, refer to the [Makefile implementation](https://github.com/openshift/operator-framework-operator-controller/blob/main/openshift/tests-extension/Makefile#L104) for proper modification procedures.