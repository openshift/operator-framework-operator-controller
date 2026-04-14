# E2E Test Isolation: Per-Scenario Catalogs via Dynamic OCI Image Building

## Problem

E2E test scenarios previously shared cluster-scoped resources (ClusterCatalogs, CRDs, packages),
causing cascading failures when one scenario left state behind. Parallelism was impossible because
scenarios conflicted on shared resource names.

## Solution

Each scenario dynamically builds and pushes its own bundle and catalog OCI images at test time,
parameterized by scenario ID. All cluster-scoped resource names include the scenario ID, making
conflicts structurally impossible.

```
Scenario starts
  -> Generate parameterized bundle manifests (CRD names, deployments, etc. include scenario ID)
  -> Build + push bundle OCI images to e2e registry via go-containerregistry
  -> Generate FBC catalog config referencing those bundle image refs
  -> Build + push catalog OCI image to e2e registry
  -> Create ClusterCatalog pointing at the catalog image
  -> Run scenario steps
  -> Cleanup all resources (including catalog)
```

### Key Properties

- Every cluster-scoped resource name includes the scenario ID -- no conflicts by construction.
- Failed scenario state is preserved for debugging without affecting other scenarios.
- Parallelism (`Concurrency > 1`) is safe without further changes.
- Adding new scenarios requires zero coordination with existing ones.

## Builder API (`test/e2e/catalog/`)

Bundles are defined as components of a catalog. A single `Build()` call builds and pushes
all bundle images, generates the FBC, and pushes the catalog image:

```go
cat := catalog.NewCatalog("test", scenarioID,
    catalog.WithPackage("test",
        catalog.Bundle("1.0.0", catalog.WithCRD(), catalog.WithDeployment(), catalog.WithConfigMap()),
        catalog.Bundle("1.2.0", catalog.WithCRD(), catalog.WithDeployment()),
        catalog.Channel("beta", catalog.Entry("1.0.0"), catalog.Entry("1.2.0")),
    ),
)
result, err := cat.Build(ctx, "v1", localRegistry, clusterRegistry)
// result.CatalogName     = "test-catalog-{scenarioID}"
// result.CatalogImageRef = "{clusterRegistry}/e2e/test-catalog-{scenarioID}:v1"
// result.PackageNames    = {"test": "test-{scenarioID}"}
```

### Bundle Options

- `WithCRD()` -- CRD with group `e2e-{id}.e2e.operatorframework.io`
- `WithDeployment()` -- Deployment named `test-operator-{id}` (includes CSV, script ConfigMap, NetworkPolicy)
- `WithConfigMap()` -- additional test ConfigMap
- `WithInstallMode(modes...)` -- sets supported install modes on the CSV
- `WithLargeCRD(fieldCount)` -- CRD with many fields for large bundle testing
- `WithClusterRegistry(host)` -- overrides the cluster-side registry host (for mirror testing)
- `StaticBundleDir(dir)` -- reads pre-built bundle manifests without parameterization (e.g. webhook-operator)
- `BadImage()` -- uses an invalid container image to trigger ImagePullBackOff
- `WithBundleProperty(type, value)` -- adds a property to bundle metadata

## Feature File Conventions

Feature files define catalogs inline via data tables:

```gherkin
Background:
  Given OLM is available
  And an image registry is available
  And a catalog "test" with packages:
    | package | version | channel | replaces | contents                   |
    | test    | 1.0.0   | alpha   |          | CRD, Deployment, ConfigMap |
    | test    | 1.0.1   | alpha   | 1.0.0    | CRD, Deployment, ConfigMap |
    | test    | 1.2.0   | beta    |          | CRD, Deployment            |
```

### Variable Substitution

Templates in feature file YAML use these variables:

| Variable | Expansion | Example |
|----------|-----------|---------|
| `${NAME}` | ClusterExtension name | `ce-abc123` |
| `${TEST_NAMESPACE}` | Scenario namespace | `ns-abc123` |
| `${SCENARIO_ID}` | Unique scenario identifier | `abc123` |
| `${PACKAGE:<name>}` | Parameterized package name | `test-abc123` |
| `${CATALOG:<name>}` | ClusterCatalog resource name | `test-catalog-abc123` |
| `${COS_NAME}` | ClusterObjectSet name | `cos-abc123` |

### Naming Conventions

| Resource | Pattern |
|----------|---------|
| CRD group | `e2e-{id}.e2e.operatorframework.io` |
| Deployment | `test-operator-{id}` |
| Package name (FBC) | `{package}-{id}` |
| Bundle image | `{registry}/bundles/{package}-{id}:v{version}` |
| Catalog image | `{registry}/e2e/{name}-catalog-{id}:{tag}` |
| ClusterCatalog | `{name}-catalog-{id}` |
| Namespace | `ns-{id}` |
| ClusterExtension | `ce-{id}` |
