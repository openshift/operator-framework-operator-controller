# OLMv1 Tests Extension
========================

This repository contains the OLMv1 tests for OpenShift.
These tests are designed to run exclusively on OpenShift clusters and are used in the OpenShift CI/CD pipeline.
They use the framework: https://github.com/openshift-eng/openshift-tests-extension

## How it works

Three things make these tests run automatically in OpenShift CI:

1. **Build** -- The [operator-controller.Dockerfile](../operator-controller.Dockerfile) builds the
   `olmv1-tests-ext` binary, gzips it, and places it at `/usr/bin/olmv1-tests-ext.gz` inside the
   operator-controller payload image.

2. **Registration** -- The [openshift/origin](https://github.com/openshift/origin) repo maps
   the image tag `olm-operator-controller` to that binary path in
   [pkg/test/extensions/binary.go](https://github.com/openshift/origin/blob/master/pkg/test/extensions/binary.go).
   This tells `openshift-tests` where to find and extract the OLMv1 test binary.

3. **Execution** -- Because the binary is inside the payload and registered in origin,
   `openshift-tests` picks it up automatically. The tests then run in release validation jobs,
   presubmit jobs for OpenShift org repositories, and periodic jobs.

These tests run across many different environments -- architectures (arm64, ppc64le, s390x),
network setups (disconnected, proxy), and topologies (SNO, bare metal, etc.) -- unless a test
opts out with a skip label like `[Skipped:Disconnected]`.

**Useful links:**

| What | Link | Description |
|------|------|-------------|
| Release jobs | [amd64.ocp.releases.ci.openshift.org](https://amd64.ocp.releases.ci.openshift.org/) | Click any build to see all validation jobs run against it |
| Component Readiness | [Sippy](https://sippy.dptools.openshift.org/sippy-ng/component_readiness/main) | Test results feed here. Failures trigger a red alert and a Slack notification to the team |
| OpenShift CI docs | [docs.ci.openshift.org](https://docs.ci.openshift.org/) | General documentation on how OpenShift CI works |
| OTE Framework | [github.com/openshift-eng/openshift-tests-extension](https://github.com/openshift-eng/openshift-tests-extension) | OpenShift Tests Extension framework - wraps Ginkgo and exposes test commands |
| OTE Enhancement | [OTE Enhancement Proposal](https://github.com/openshift/enhancements/blob/master/enhancements/testing/openshift-tests-extension.md) | Official design doc for the OpenShift Tests Extension framework |
| Ginkgo v2 docs | [onsi.github.io/ginkgo](https://onsi.github.io/ginkgo/) | Official Ginkgo BDD testing framework documentation |
| Ginkgo CLI reference | [Ginkgo CLI flags](https://onsi.github.io/ginkgo/#the-ginkgo-cli) | Complete reference for Ginkgo command-line flags and options |
| Help with alerts | `#forum-ocp-testplatform` on Slack | Managed by the TRT team |
| Help with OTE | `#wg-openshift-tests-extension` on Slack | Questions about the OpenShift Tests Extension framework |

## Design Architecture

This extension has two categories of tests. They differ in **where they run** in CI:

### Standard tests (`test/`)

Written by the OLMv1 development team. These run in **all** OpenShift CI jobs that use the
`olmv1/*` suites -- release validation, presubmit, and periodic jobs.

Each suite maps to a parent suite in `openshift-tests` (defined in [`cmd/main.go`](cmd/main.go)):

| Suite | Runs inside | What it includes |
|-------|-------------|------------------|
| `olmv1/parallel` | `openshift/conformance/parallel` | All tests except `[Serial]` and `[Slow]` |
| `olmv1/serial` | `openshift/conformance/serial` | `[Serial]` tests (excludes `[Disruptive]` and `[Slow]`) |
| `olmv1/slow` | `openshift/optional/slow` | `[Slow]` tests only |
| `olmv1/all` | *(standalone)* | Everything |

These suites also pick up `Extended + ReleaseGate` tests from `test/qe/` (see below).

### Extended tests (`test/qe/`)

Migrated from the QE tests-private repository. The framework auto-labels every test under
`test/qe/specs/` as `Extended` (see [`cmd/main.go`](cmd/main.go)).

Whether a test also has the `ReleaseGate` label decides where it runs:

| Labels | Runs in standard suites? | Runs in extended suites? | Where it runs in CI |
|--------|--------------------------|--------------------------|---------------------|
| `Extended` + `ReleaseGate` | Yes | Yes | Everywhere (release, presubmit, periodic) |
| `Extended` only | No | Yes | QE periodic jobs only |

The filter logic is in [`test/qe/util/filters/filters.go`](test/qe/util/filters/filters.go).

The extended suites break down like this:

```text
olmv1/extended                        # All Extended tests
├── extended/releasegate              # Extended + ReleaseGate (also in standard suites)
└── extended/candidate                # Extended without ReleaseGate
    ├── candidate/function            # Functional tests (no StressTest)
    │   ├── candidate/parallel        # No [Serial], no [Slow]
    │   ├── candidate/serial          # [Serial] only
    │   ├── candidate/fast            # parallel + serial (no [Slow])
    │   └── candidate/slow            # [Slow] only
    └── candidate/stress              # StressTest label
```

## QE Periodic Jobs

The QE periodic jobs live in the [openshift/release](https://github.com/openshift/release) repo at
[ci-operator/config/openshift/operator-framework-operator-controller/](https://github.com/openshift/release/tree/master/ci-operator/config/openshift/operator-framework-operator-controller)
(look for files ending in `__periodics.yaml`).

Example ([source](https://github.com/openshift/release/blob/main/ci-operator/config/openshift/operator-framework-operator-controller/openshift-operator-framework-operator-controller-release-4.22__periodics.yaml#L112-L120)):

```yaml
- as: e2e-aws-ovn-techpreview-extended-f1
  cron: 2 10 * * *
  steps:
    cluster_profile: openshift-org-aws
    env:
      FEATURE_SET: TechPreviewNoUpgrade
      TEST_ARGS: --monitor=watch-namespaces
      TEST_SUITE: olmv1/extended/candidate/fast
    workflow: openshift-e2e-aws-ovn
```

## How to Run the Tests Locally

You must run OTE tests (`./bin/olmv1-tests-ext`) against an OCP Cluster with TechPreview Features enabled.

### Setup: Get an OpenShift Cluster

Use Cluster Bot to create an OpenShift cluster with OLMv1 installed:

```shell
launch 4.20 gcp,techpreview
```

Set `KUBECONFIG`:

```shell
mv ~/Downloads/cluster-bot-2025-08-06-082741.kubeconfig ~/.kube/cluster-bot.kubeconfig
export KUBECONFIG=~/.kube/cluster-bot.kubeconfig
```

### Two Ways to Run Tests

#### 1. **Developer-Friendly Output** (For local development)

Use the local dev commands (`run-suite-dev`, `run-test-dev`) that provide clean, human-readable output:

**Implementation:** Local dev commands in `localdevoutput/` are excluded from production builds using Go build tags. Only included with `make build-local-dev`. See [localdevoutput/README.md](localdevoutput/README.md).

| Command | Description |
|---------|-------------|
| `make build-local-dev` | Builds the test binary with local dev commands |
| `make test-local SUITE=olmv1/all` | Runs a test suite with clean, color-coded output |
| `make test-local-single TEST="test name"` | Runs a single test with clean output |
| `make list-test-names` | Lists all available test names |

**Example**

```bash
export KUBECONFIG=~/.kube/cluster-bot.kubeconfig
make build-local-dev
make test-local SUITE=olmv1/all
```

**Output:** Clean, color-coded summary with live progress:
```text
[46/46] ▶ Running: [sig-olmv1][OCPFeatureGate:NewOLMWebhookProviderOpenshiftServiceCA] OLMv1 operator with webhooks should have a working validating webhook
  ✓ PASSED [194.1 seconds] (Total: ✓45 ✗0)


════════════════════════════════════════════════════════
  Final Summary
════════════════════════════════════════════════════════
✓ Passed:  45
✗ Failed:  0
⊘ Skipped: 1

✓ ALL TESTS PASSED!
```

#### 2. **Raw OTE Framework Output** (For CI/CD integration)

Run the binary directly for structured JSON reports:

| Command | Description |
|---------|-------------|
| `./bin/olmv1-tests-ext info` | Shows info about the test binary and registered test suites |
| `./bin/olmv1-tests-ext list` | Lists all available test cases |
| `./bin/olmv1-tests-ext run-suite olmv1/all` | Runs the full OLMv1 test suite with JSON output |
| `./bin/olmv1-tests-ext run-test -n <test-name>` | Runs one specific test with JSON output |

**Example:**

```bash
export KUBECONFIG=~/.kube/cluster-bot.kubeconfig
make build
./bin/olmv1-tests-ext run-suite olmv1/all
```

**Output:** Structured JSON report (as used by Component Readiness and other integrated solutions):
```text
Running Suite:  - /Users/camilam/go/src/github/operator-framework-operator-controller/openshift/tests-extension
===============================================================================================================
Random Seed: 1753508546 - will randomize all specs

Will run 1 of 1 specs
------------------------------
[sig-olmv1] OLMv1 should pass a trivial sanity check
/Users/camilam/go/src/github/operator-framework-operator-controller/openshift/tests-extension/test/olmv1.go:26
• [0.000 seconds]
------------------------------

Ran 1 of 1 Specs in 0.000 seconds
SUCCESS! -- 1 Passed | 0 Failed | 0 Pending | 0 Skipped
[
  {
    "name": "[sig-olmv1] OLMv1 should pass a trivial sanity check",
    "lifecycle": "blocking",
    "duration": 0,
    "startTime": "2025-07-26 05:42:26.553852 UTC",
    "endTime": "2025-07-26 05:42:26.580263 UTC",
    "result": "passed",
    "output": ""
  }
]
```

This is the same output format used by:
- **Component Readiness** ([Sippy](https://sippy.dptools.openshift.org/sippy-ng/component_readiness/main))
- **OpenShift CI/CD** pipeline
- **Release validation jobs**
- Any automated test processing tools

**When to use which:**
- Use **clean output** (`make test-local` or `make test-local-single`) for local development, debugging, and quick visual feedback
- Use **raw output** (direct binary execution with `./bin/olmv1-tests-ext`) when you need JSON reports, CI/CD integration, or programmatic processing

### Discovering Available Flags

The OTE framework wraps Ginkgo and exposes its own set of commands and flags. To see what's available:

```bash
# See all available commands
./bin/olmv1-tests-ext --help

# See flags for running test suites
./bin/olmv1-tests-ext run-suite --help

# See flags for running individual tests
./bin/olmv1-tests-ext run-test --help
```

**Available OTE-specific flags:**
- `--component string` - Specify the component to enable (default "default")
- `--max-concurrency int` - Maximum number of tests to run in parallel (default 10)
- `--output string` - Output mode (default "json")
- `--junit-path string` - Write results to JUnit XML (for `run-suite`)
- `--names stringArray` - Specify test name, can be used multiple times (for `run-test`)

**Note:** The OTE framework does not expose all Ginkgo CLI flags.
It provides a simplified interface focused on running tests in OpenShift environments.
For full Ginkgo flag reference, see the [Ginkgo CLI documentation](https://onsi.github.io/ginkgo/#the-ginkgo-cli).

## Development Workflow

- Add or update tests in: `openshift/tests-extension/tests/`
- Run `make build` to build the test binary
- You can run the full suite or one test using the commands in the table above
- Before committing your changes:
    - Run `make update-metadata` or `make build-update`
    - Run `make verify` to check formatting, linting, and validation

**IMPORTANT** Ensure that you either test any new test with `/payload-aggregate`
to avoid issues with Sippy or other tools due flakes. Run at least 5 times.

**Examples**

- For `[Serial]` tests run: `/payload-aggregate periodic-ci-openshift-release-master-ci-4.20-e2e-gcp-ovn-techpreview-serial 5`
- For others run: `/payload-aggregate periodic-ci-openshift-release-master-ci-4.20-e2e-gcp-ovn-techpreview 5`

## How to Rename a Test

1. Run `make list-test-names` to see the current test names
2. Find the name of the test you want to rename
3. Add a Ginkgo label with the original name, like this:

```go
It("should pass a renamed sanity check",
	Label("original-name:[sig-olmv1] OLMv1 should pass a trivial sanity check"),
	func(ctx context.Context) {
		Expect(len("test")).To(BeNumerically(">", 0))
	})
```

4. Run `make build-update` to update the metadata

**Note:** Only add the label once. Do not update it again after future renames.

## How to Delete a Test

1. Run `make list-test-names` to find the test name
2. Add the test name to the `IgnoreObsoleteTests` block in `main.go`, like this:

```go
ext.IgnoreObsoleteTests(
    "[sig-olmv1] My removed test name",
)
```

3. Delete the test code from your suite (like `olmv1.go`)
4. Run `make build-update` to clean the metadata

**WARNING**: Deleting a test may cause issues with Sippy https://sippy.dptools.openshift.org/sippy-ng/
or other tools that expected the Unique TestID tracked outside of this repository. [More info](https://github.com/openshift-eng/ci-test-mapping)
Check the status of https://issues.redhat.com/browse/TRT-2208 before proceeding with test deletions.

## Presubmit CI Jobs

Tests are configured in: [ci-operator/config/openshift/operator-framework-operator-controller](https://github.com/openshift/release/blob/master/ci-operator/config/openshift/operator-framework-operator-controller/)

Every PR to `operator-framework-operator-controller` triggers presubmit jobs defined in the
[main branch config](https://github.com/openshift/release/blob/master/ci-operator/config/openshift/operator-framework-operator-controller/openshift-operator-framework-operator-controller-main.yaml).

These jobs run the `olmv1/all` suite (which includes standard tests **and** `Extended + ReleaseGate`
tests) against a freshly built OpenShift release that includes the PR's images.

Example ([source](https://github.com/openshift/release/blob/master/ci-operator/config/openshift/operator-framework-operator-controller/openshift-operator-framework-operator-controller-main.yaml)):

```yaml
- as: e2e-aws-techpreview-olmv1-ext
  steps:
    cluster_profile: aws-3
    env:
      FEATURE_SET: TechPreviewNoUpgrade
      # Only enable 'watch-namespaces' monitor to avoid job failures from other default monitors 
      # in openshift-tests (like apiserver checks, alert summaries, etc). In this job, the selected 
      # OLMv1 test passed, but the job failed because a default monitor failed. 
      #
      # 'watch-namespaces' is very lightweight and rarely fails, so it's a safe choice.
      # There is no way to fully disable all monitors, but we can use this option to reduce noise.
      #
      # See: ./openshift-tests run --help (option: --monitor)
      TEST_ARGS: --monitor=watch-namespaces
      TEST_SUITE: olmv1/all
    test:
    - ref: openshift-e2e-test
    workflow: openshift-e2e-aws

- as: e2e-aws-olmv1-ext
  steps:
    cluster_profile: aws-3
    env:
      # Only enable 'watch-namespaces' monitor to avoid job failures from other default monitors 
      # in openshift-tests (like apiserver checks, alert summaries, etc). In this job, the selected 
      # OLMv1 test passed, but the job failed because a default monitor failed. 
      #
      # 'watch-namespaces' is very lightweight and rarely fails, so it's a safe choice.
      # There is no way to fully disable all monitors, but we can use this option to reduce noise.
      #
      # See: ./openshift-tests run --help (option: --monitor)
      TEST_ARGS: --monitor=watch-namespaces
      TEST_SUITE: olmv1/all
    test:
    - ref: openshift-e2e-test
    workflow: openshift-e2e-aws
```

This works because `include_built_images: true` in the release config injects the PR's freshly
built images into the test cluster. More info:
[Testing with an ephemeral OpenShift release](https://docs.ci.openshift.org/docs/architecture/ci-operator/#testing-with-an-ephemeral-openshift-release).

There is also a `tests-extension` sanity job that runs only when files under
`openshift/tests-extension/` change. It verifies formatting, builds the binary, and checks
that the metadata is up to date:

```yaml
- as: tests-extension
  run_if_changed: ^(openshift/tests-extension/)
  steps:
    test:
    - as: sanity
      commands: |
        cd openshift/tests-extension
        make verify
        make build
        make verify-metadata
      from: src
```

## Makefile Commands

| Target                           | Description                                                                  |
|----------------------------------|------------------------------------------------------------------------------|
| `make build`                     | Builds the test binary.                                                      |
| `make test-local SUITE=<suite>`  | Runs a test suite with clean, human-readable output for local development. |
| `make test-local-single TEST="<name>"` | Runs a single test with clean, human-readable output. |
| `make list-test-names`           | Shows all test names in the binary.                                          |
| `make update-metadata`           | Updates the metadata JSON file.                                              |
| `make build-update`              | Runs build + update-metadata + cleans codeLocations.                         |
| `make verify`                    | Runs formatting, vet, and linter.                                            |
| `make clean-metadata`            | Removes machine-specific codeLocations from the JSON metadata. [More info](https://issues.redhat.com/browse/TRT-2186) |

**Note:** Metadata is stored in: `.openshift-tests-extension/openshift_payload_olmv1.json`

## FAQ

### Why don't we have a Dockerfile for `olmv1-tests-ext`?

We do not provide a Dockerfile for `olmv1-tests-ext` because building and shipping a 
standalone image for this test binary would introduce unnecessary complexity.

Technically, it is possible to create a new OpenShift component just for the 
OLMv1 tests and add a corresponding test image to the payload. However, doing so requires 
onboarding a new component, setting up build pipelines, and maintaining image promotion 
and test configuration — all of which adds overhead.

From the OpenShift architecture point of view:

1. Tests for payload components are part of the product. Many users (such as storage vendors, or third-party CNIs)
rely on these tests to validate that their solutions are compatible and conformant with OpenShift.

2. Adding new images to the payload comes with significant overhead and cost. 
It is generally preferred to include tests in the same image as the component 
being tested whenever possible.

### Why do we need to run `make update-metadata`?

Running `make update-metadata` ensures that each test gets a unique and stable **TestID** over time.

The TestID is used to identify tests across the OpenShift CI/CD pipeline and reporting tools like Sippy. 
It helps track test results, detect regressions, and ensures the correct tests are 
executed and reported.

This step is important whenever you add, rename, or delete a test.
More information:
- https://github.com/openshift/enhancements/blob/master/enhancements/testing/openshift-tests-extension.md#test-id
- https://github.com/openshift-eng/ci-test-mapping

### How to get help with OTE?

For help with the OpenShift Tests Extension (OTE), you can:
#wg-openshift-tests-extension

### How to update the go.mod/go.sum files with replaces?

To get the latest replaces for ocp/* modules, run the following command:

```shell
$ ./hack/ocp-replace.sh 
Discovering latest OCP commit from https://github.com/openshift/kubernetes.git…
Resolving pseudo-version for k8s.io/kubernetes@891f5bb0306166d5625b89fc8dc86bbc8c85f549…
Resolved OCP version: v0.0.0-20251108023427-891f5bb03061
Updating go.mod replaces…

Done.
  OCP commit:  891f5bb0306166d5625b89fc8dc86bbc8c85f549
  OCP version: v0.0.0-20251108023427-891f5bb03061
go.mod and go.sum vendor/ updated.
```
