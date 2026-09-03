# OLM inventory and status metrics spike

## Goal

Prototype kube-state-metrics-style metrics for the current catalog and extension state. The spike intentionally models the useful data first; observed series count and churn will determine what is suitable for long-term in-cluster use or a separate bounded Telemeter metric.

The metrics are generated on scrape from each component's controller-runtime cache. They are not updated from reconcile code, so deleted objects and obsolete label sets disappear from subsequent collections without explicit cleanup.

A cache list failure is returned as an invalid Prometheus metric rather than silently reporting an empty inventory.

## ClusterExtension metrics

Operator-controller exposes these metrics.

### `olm_clusterextension_info`

One gauge with value `1` is emitted per `ClusterExtension`:

```text
olm_clusterextension_info{
  name="example",
  package="example-operator",
  channels="preview,stable",
  catalog="community-operators",
  installed_version="1.2.3"
} 1
```

| Label | Source | Empty value |
| --- | --- | --- |
| `name` | `.metadata.name` | never |
| `package` | `.spec.source.catalog.packageName` | no catalog source |
| `channels` | sorted `.spec.source.catalog.channels`, comma-joined | no channel constraint |
| `catalog` | `.spec.source.catalog.selector.matchLabels[MetadataNameLabel]` | selector does not pin one named catalog |
| `installed_version` | `.status.install.bundle.version` | nothing installed |

`name` is retained as the technical series identity. Two `ClusterExtension` objects can request the same package and otherwise have identical labels; omitting `name` would make the collector emit duplicate Prometheus series.

Channels remain one deterministic label value so the info metric emits one active series per extension. This avoids multiplying series by the number of requested channels, at the cost of treating every channel combination as its own label value.

### `olm_clusterextension_condition`

One gauge with value `1` is emitted for each present `Installed` and `Progressing` condition:

```text
olm_clusterextension_condition{
  package="example-operator",
  name="example",
  type="Progressing",
  status="False",
  reason="Blocked"
} 1
```

Raw condition values are retained instead of deriving a lossy health classification. This distinguishes a healthy installed version that is retrying an update from an extension with no successful installation.

Keeping conditions separate prevents every readiness transition from also creating a new inventory-series combination. An absent condition emits no condition series.

## ClusterCatalog metrics

Catalogd exposes these metrics because it is authoritative for the content being served.

### `olm_cluster_catalog_serving`

One gauge is emitted per `ClusterCatalog`:

```text
olm_cluster_catalog_serving{
  name="community-operators",
  digest="sha256:e3b0c44298fc...",
  reason="Available"
} 1
```

| State | Value | `digest` | `reason` |
| --- | ---: | --- | --- |
| `Serving=True` | `1` | Digest extracted from `.status.resolvedSource.image.ref` | Serving condition reason |
| `Serving=False` or `Unknown` | `0` | empty | Serving condition reason |
| Serving condition absent | `0` | empty | empty |

The digest identifies the complete catalog snapshot, not an individual package. A new catalog image can add or remove versions for one package or change many packages at once.

During a failed catalog refresh, catalogd continues serving the previous content. The metric therefore remains `1` with the previous digest while the `Progressing` condition reports the refresh failure. After a successful switch, the old digest label set disappears from new scrapes and a series with the new digest appears. Prometheus marks the old series stale but retains its historical samples.

Deleting a `ClusterCatalog` removes its series; deletion does not emit a final zero.

### `olm_cluster_catalog_condition`

One gauge with value `1` is emitted for each present `Serving` and `Progressing` condition:

```text
olm_cluster_catalog_condition{
  name="community-operators",
  type="Progressing",
  status="True",
  reason="Retrying"
} 1
```

This condition metric explains states that the serving metric intentionally does not, including a failed attempt to replace catalog content that is still available to clients.

## Catalog selection limitation

`ClusterExtension` does not record the catalog that supplied its installed bundle. Its `catalog` label is truthful only when the source selector has an exact `MetadataNameLabel` match. It is empty for broad or expression-based selectors and must not claim that the selector is the resolved source.

The resolver retains the catalog name alongside each candidate and selects a single winning bundle, but its current result discards the catalog name. Truthfully answering "which catalog supplied this installed bundle?" requires returning the winning catalog and recording it in `ClusterExtension` status.

That is an API/status change, not a metric-layer tweak. It requires upstream API review and the normal generated-code, CRD, reference-documentation, and API-diff workflow. It is outside this prototype.

## Updates and transitions

The metrics describe current state:

```text
catalog digest changes
    -> new package metadata becomes available
    -> ClusterExtension resolves a bundle
    -> installed version changes
```

They can show the catalog digest and installed version before and after an update, but they cannot truthfully encode a durable event containing `from version`, `to version`, transition time, and outcome. A reconcile-time counter would reset with the process and create unbounded version-pair labels. Durable update transitions remain an Events, audit, or Insights/CCX concern.

## Cardinality and churn

For `N` extensions and `C` catalogs, the maximum active series represented here is approximately:

```text
N extension info
+ 2N extension conditions
+ C catalog serving
+ 2C catalog conditions
= 3N + 3C
```

Missing conditions reduce the actual count. Label values do not form a cross-product because the collectors emit concrete object and condition records.

Prometheus creates a new historical time series whenever any label changes. Expected churn comes from installed versions, requested channel sets, resolved catalog digests, and condition transitions. Removing constant labels reduces payload size but does not reduce active series while `name` remains unique; removing changing labels reduces historical churn. A Telemeter-safe design will require bounded aggregation rather than exporting these identity-rich series.

## Local visualization

After deploying the prototype with `make run`, run:

```shell
uv run --script hack/watch-clusterextension-metrics.py
```

The terminal view follows all four spike metrics, scraping extension metrics from operator-controller and catalog metrics from catalogd. Each metric has independent active-series, cumulative-series, gauge-value, sparkline, and per-label cardinality output. A label change increases that metric's cumulative count even when its active count stays constant. History is process-local and resets when the script exits.

Use `--once` for one scrape, `--interval` to change the polling period, and `--self-test` to check the exposition parser. The script obtains an operator-controller service-account token and manages local port-forwards for operator-controller on `18443` and catalogd on `17443`; use `--port` or `--catalog-port` if either port is occupied.

## Evaluation

Use a representative cluster to measure:

1. active series per metric;
2. cumulative series churn during catalog refreshes, installs, upgrades, failures, and recovery;
3. distinct and changing values per label;
4. which labels answer actual inventory and operational questions.

Then remove labels or split out bounded aggregate metrics based on evidence rather than treating this spike's contracts as stable APIs.
