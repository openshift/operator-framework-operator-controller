Default Catalog Consistency
========================

These tests verify the consistency of the default catalogs shipped with OpenShift.
Default catalogs provide the operator content that OLM manages on every cluster.
If a catalog is broken, operators cannot be installed or upgraded, which directly
affects supportability.

## What it checks

The test suite reads catalog image references from the ClusterCatalog manifests
(at `catalogd/manifests.yaml`), pulls each image, and runs four layers of checks:

| Layer | What it validates | Examples |
|-------|-------------------|----------|
| **Image** | The container image is well-formed | Valid OCI manifest, has required labels (e.g. `operators.operatorframework.io.index.configs.v1=/configs`) |
| **Multi-arch** | The image supports all required platforms | linux/amd64, linux/arm64, linux/ppc64le, linux/s390x |
| **Filesystem** | Expected files and directories exist inside the image | `bin/opm` is a valid Go binary, `configs/` and `tmp/cache/` directories exist |
| **Catalog (FBC)** | The File-Based Catalog content is valid | No bundles use the deprecated `olm.bundle.object` property; all channel heads have `olm.csv.metadata` |

The check implementations live in `pkg/check/` and the test entry point is
`test/validate/suite_test.go`.

## CI configuration

CI and periodic jobs are configured in the
[openshift/release](https://github.com/openshift/release) repository under
[ci-operator/config/openshift/operator-framework-operator-controller/](https://github.com/openshift/release/tree/master/ci-operator/config/openshift/operator-framework-operator-controller).

### Presubmit

A presubmit job runs on every PR that touches catalog-related files.
See the [main branch config](https://github.com/openshift/release/blob/master/ci-operator/config/openshift/operator-framework-operator-controller/openshift-operator-framework-operator-controller-main.yaml)
(search for `default-catalog-consistency`).

### Periodic

Each release branch has a periodic job that runs on a schedule and reports results
to [Sippy](https://sippy.dptools.openshift.org/sippy-ng/). Look for files ending
in `__periodics.yaml`, for example:

- [release-4.20 periodics](https://github.com/openshift/release/blob/main/ci-operator/config/openshift/operator-framework-operator-controller/openshift-operator-framework-operator-controller-release-4.20__periodics.yaml)

You can view results per release in Sippy. For example:
[4.22 default-catalog jobs](https://sippy.dptools.openshift.org/sippy-ng/jobs/4.22?filters=%257B%2522items%2522%253A%255B%257B%2522columnField%2522%253A%2522variants%2522%252C%2522operatorValue%2522%253A%2522has%2520entry%2522%252C%2522value%2522%253A%2522never-stable%2522%252C%2522not%2522%253Atrue%257D%252C%257B%2522id%2522%253A99%252C%2522columnField%2522%253A%2522name%2522%252C%2522operatorValue%2522%253A%2522contains%2522%252C%2522value%2522%253A%2522default-catalog%2522%257D%255D%257D&sort=asc&sortField=net_improvement).

## Monitoring and alerts

When periodic jobs fail, alerts are sent to the `#forum-ocp-catalogs-program` Slack channel.
[Example alert](https://redhat-internal.slack.com/archives/C08D3G4EMRA/p1777089324894579).

> **Note:** Currently, thresholds are high and many alerts are caused by flakes or
> infrastructure issues (like the example above). We must still monitor these alerts
> because a real failure means the default catalogs are broken -- operators cannot be
> installed or upgraded, and this is critical for the content managed by OLM.

## Running tests locally

```bash
make test-catalog
```

## Before pushing changes

Run the following to verify that tests pass and code is properly formatted:

```bash
make verify
```
