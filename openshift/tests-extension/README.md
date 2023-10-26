# OLMv1 Tests Extension
========================

This repository contains the OLMv1 tests for OpenShift.
These tests run against OpenShift clusters and are meant to be used in the OpenShift CI/CD pipeline.
They use the framework: https://github.com/openshift-eng/openshift-tests-extension

## How to Run the Tests Locally

| Command                                         | Description                                                              |
|-------------------------------------------------|--------------------------------------------------------------------------|
| `make build`                                    | Builds the OLMv1 test binary.                                            |
| `./bin/olmv1-tests-ext info`                    | Shows info about the test binary and registered test suites.             |
| `./bin/olmv1-tests-ext list`                    | Lists all available test cases.                                          |
| `./bin/olmv1-tests-ext run-suite olmv1/all`     | Runs the full OLMv1 test suite.                                          |
| `./bin/olmv1-tests-ext run-test -n <test-name>` | Runs one specific test. Replace <test-name> with the test's full name.   |


## How to Run the Tests Locally

The tests can be run locally using the `olmv1-tests-ext` binary against an OpenShift cluster or a vanilla Kubernetes cluster.
Features and checks which are OpenShift-specific will be skipped when running against a vanilla Kubernetes cluster.

Use the environment variable `KUBECONFIG` to point to your cluster configuration file such as:

```shell
KUBECONFIG=path/to/kubeconfig ./bin/olmv1-tests-ext run-test -n <test-name>
```

To run tests that include tech preview features, 
you need a cluster with OLMv1 installed and those features enabled.

### Local Test using OLMv1 and OCP

1. Use the `Cluster Bot` to create an OpenShift cluster with OLMv1 installed.

**Example:**

```shell
launch 4.20 gcp,techpreview
```

2. Set the `KUBECONFIG` environment variable to point to your OpenShift cluster configuration file.

**Example:**

```shell
mv ~/Downloads/cluster-bot-2025-08-06-082741.kubeconfig ~/.kube/cluster-bot.kubeconfig
export KUBECONFIG=~/.kube/cluster-bot.kubeconfig
```

3. Run the tests using the `olmv1-tests-ext` binary.

**Example:**
```shell
./bin/olmv1-tests-ext run-suite olmv1/all
```

### Local Test using OLMv1 and Kind

**Prerequisites:**

Install OLMv1 before running the tests. 
You can use the `kind` tool to create a local Kubernetes cluster with OLMv1 installed.
Furthermore, if you are using feature gates in your test you will need to
ensure that those are enabled in your cluster.

**Example:**

export KUBECONFIG=$HOME/.kube/config

```shell
$ ./bin/olmv1-tests-ext run-suite olmv1/all
  Running Suite:  - /Users/camilam/go/src/github/operator-framework-operator-controller/openshift/tests-extension
  ===============================================================================================================
  Random Seed: 1753508546 - will randomize all specs

  Will run 1 of 1 specs
  ------------------------------
  [sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 CRDs should verify required CRDs are installed and active
  /Users/camilam/go/src/github/operator-framework-operator-controller/openshift/tests-extension/test/olmv1.go:37
  [INFO] [env] Using kubeconfig: /Users/camilam/.kube/config[WARN] Skipping feature capability check: not OpenShift  STEP: verifying CRD clusterextensions.olm.operatorframework.io @ 07/26/25 06:42:26.57
    STEP: verifying CRD clustercatalogs.olm.operatorframework.io @ 07/26/25 06:42:26.575
  • [0.026 seconds]
  ------------------------------

  Ran 1 of 1 Specs in 0.026 seconds
  SUCCESS! -- 1 Passed | 0 Failed | 0 Pending | 0 Skipped
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
    "name": "[sig-olmv1][OCPFeatureGate:NewOLM] OLMv1 CRDs should verify required CRDs are installed and active",
    "lifecycle": "blocking",
    "duration": 26,
    "startTime": "2025-07-26 05:42:26.553886 UTC",
    "endTime": "2025-07-26 05:42:26.580057 UTC",
    "result": "passed",
    "output": "[INFO] [env] Using kubeconfig: /Users/camilam/.kube/config[WARN] Skipping feature capability check: not OpenShift  STEP: verifying CRD clusterextensions.olm.operatorframework.io @ 07/26/25 06:42:26.57\n  STEP: verifying CRD clustercatalogs.olm.operatorframework.io @ 07/26/25 06:42:26.575\n"
  },
  {
    "name": "[sig-olmv1] OLMv1 should pass a trivial sanity check",
    "lifecycle": "blocking",
    "duration": 26,
    "startTime": "2025-07-26 05:42:26.553852 UTC",
    "endTime": "2025-07-26 05:42:26.580263 UTC",
    "result": "passed",
    "output": ""
  }
]
```

## Writing Tests

You can write tests in the `openshift/tests-extension/tests/` directory. 
Please follow these guidelines:

1. Skip OpenShift-specific logic on vanilla Kubernetes

If your test requires OpenShift-only APIs (e.g., clusterversions.config.openshift.io), 
guard it using `env.Get().IsOpenShift` to ensure it skips gracefully when running 
on vanilla Kubernetes clusters:

```go
    if !env.Get().IsOpenShift {
        extlogs.Warn("Skipping test: not running on OpenShift")
        Skip("This test requires OpenShift APIs")
    }
```

Or, if used within helper functions:
```go
    if !env.Get().IsOpenShift {
        extlogs.Warn("Skipping feature capability check: not OpenShift")
        return
    }
```

This ensures compatibility when running tests in non-OpenShift environments such as KinD.

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

## E2E Test Configuration

Tests are configured in: [ci-operator/config/openshift/operator-framework-operator-controller](https://github.com/openshift/release/blob/master/ci-operator/config/openshift/operator-framework-operator-controller/)

Here is a CI job example:

```yaml
- as: e2e-aws-techpreview-olmv1-ext
  steps:
    cluster_profile: aws
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
```

This uses the `openshift-tests` binary to run OLMv1 tests against a test OpenShift release.

It works for pull request testing because of this:

```yaml
releases:
  latest:
    integration:
      include_built_images: true
```

More info: https://docs.ci.openshift.org/docs/architecture/ci-operator/#testing-with-an-ephemeral-openshift-release

## Makefile Commands

| Target                   | Description                                                                  |
|--------------------------|------------------------------------------------------------------------------|
| `make build`             | Builds the test binary.                                                      |
| `make update-metadata`   | Updates the metadata JSON file.                                              |
| `make build-update`      | Runs build + update-metadata + cleans codeLocations.                         |
| `make verify`            | Runs formatting, vet, and linter.                                            |
| `make list-test-names`   | Shows all test names in the binary.                                          |
| `make clean-metadata`    | Removes machine-specific codeLocations from the JSON metadata. [More info](https://issues.redhat.com/browse/TRT-2186) |

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