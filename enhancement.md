---
title: olm-v1-metrics
authors:
  - @theobarberbany
reviewers:
  - @joelanford
approvers: []
api-approvers:
  - @joelspeed
creation-date: 2026-09-04
last-updated: 2026-09-10
status: provisional
tracking-link:
  - https://issues.redhat.com/browse/OCPSTRAT-3069
---

# OLM v1 Metrics

## Summary

OLM v1 does not currently provide the fleet inventory and health data available
for OLM v0, or the near-real-time catalog signals needed to investigate
incidents such as HPSTRAT-277. This enhancement makes detailed OLM v1 package,
version, resolved catalog, configured channel constraints, and condition data
available through Insights with approximately two-hour freshness, while
forwarding only bounded catalog digest and aggregate ClusterExtension health
series through Telemeter for approximately five-minute incident detection. It
also adds resolved catalog attribution to ClusterExtension status and exposes
detailed in-cluster metrics
for cluster-level troubleshooting, providing the required visibility without
exceeding the Telemeter cardinality budget.

## Motivation

OLM v1 resources are not represented in existing OLM v0 fleet reporting, so
adoption, version distribution, and health cannot be measured consistently.

Product analysis needs detailed data with approximately two-hour freshness;
SRE incident detection needs bounded signals within approximately five minutes.
Detailed dimensions belong in Insights, while only catalog digest and aggregate
health belong in Telemeter.

ClusterExtension status must also record the catalog selected for the installed
bundle. Channel analysis uses the requested constraints from the specification
because OLM v1 does not select a single channel during resolution.

### User Stories

**Product managers** need OLM v1 inventory and health data to measure adoption,
composition, versions, upgrades, and migration from OLM v0.

**SREs** need to correlate catalog digest changes with abnormal extensions to
identify the blast radius of a bad catalog push.

**Cluster administrators and support engineers** need detailed in-cluster state
to diagnose affected extensions and catalogs.

### Goals

- Provide detailed OLM v1 inventory and health data through Insights.
- Provide bounded catalog and aggregate health signals through Telemeter.
- Expose detailed ClusterExtension and ClusterCatalog metrics in-cluster.
- Record the resolved catalog for each installed extension.

### Non-Goals

- Building Insights/CCX reports or lifecycle-data joins.
- Sending per-extension, package, version, channel, or reason data through Telemeter.
- Recording durable update lineage or adding an update-event counter.

## Proposal

Deliver four linked changes:

1. Add resolved catalog attribution to `ClusterExtension.status`.
2. Expose detailed ClusterExtension and ClusterCatalog metrics in-cluster, plus
   bounded total and abnormal recording rules.
3. Add OLM v1 ClusterExtension and ClusterCatalog data to the Insights archive.
4. Allowlist only catalog serving `{name,digest}`, total, and abnormal series
   for Telemeter.

### Workflow Description

Operator-controller and catalogd expose current resource state as metrics.
Insights gathers detailed inventory approximately every two hours. Cluster
monitoring evaluates the bounded recording rules and forwards the three
allowlisted series to Telemeter approximately every five minutes.

Product analysis uses Insights, SRE fleet investigation uses Telemeter, and
cluster administrators use the detailed in-cluster metrics.

### API Extensions

`ClusterExtension.status.install` currently identifies only the installed bundle's
name, version, and optional release. It will also record the catalog selected for
that bundle. Channels remain zero-or-more requested resolution constraints in
the specification; OLM v1 does not select a single channel.

This is an additive, controller-populated status change. It does not change the
specification, admission, or finalizers. During a failed upgrade, the catalog
continues to describe the bundle that remains installed.

### Topology Considerations

#### Hypershift / Hosted Control Planes

OLM v1 does not currently support hosted control plane clusters. This proposal
does not add that support.

#### Standalone Clusters

The API fields and in-cluster metrics apply normally. Insights and Telemeter
data is available on connected clusters using the existing remote-health paths.

#### Single-node Deployments or MicroShift

SNO uses the same collectors and bounded recording rules without additional
components. Resource use scales with the number of ClusterExtensions and
ClusterCatalogs.

OLM v1 does not currently support MicroShift. This proposal does not add that
support.

#### OpenShift Kubernetes Engine

The change applies where OLM v1 and the existing cluster monitoring and
remote-health capabilities are available. It adds no OKE-specific behavior.

### Implementation Details/Notes/Constraints

#### Data routing

| Ask | Path | Freshness |
| --- | --- | --- |
| Adoption, packages, versions, configured channel constraints, resolved catalogs, detailed health, installed-version and condition snapshots, and lifecycle inputs | Insights | Approximately two hours |
| Cluster troubleshooting | In-cluster metrics | Current scrape |
| Fleet aggregate health | Insights and bounded Telemeter aggregates | Two hours / five minutes |
| Catalog digest blast radius and digest-change correlation | Telemeter | Approximately five minutes |

#### Metrics data model

```text
olm_clusterextension_info{
  name="example",
  package="example-operator",
  channels="stable",
  catalog="community-operators",
  installed_version="1.2.3"
} 1

olm_clusterextension_condition{
  name="example",
  package="example-operator",
  type="Progressing",
  status="False",
  reason="Blocked"
} 1

olm_cluster_catalog_serving{
  name="community-operators",
  digest="sha256:abc123",
  reason="Available"
} 1

olm_cluster_catalog_condition{
  name="community-operators",
  type="Progressing",
  status="True",
  reason="Retrying"
} 1

cluster:olm_clusterextension_total 12
cluster:olm_clusterextension_abnormal 2
```

The `channels` label contains the sorted, comma-separated constraints from
`ClusterExtension.spec.source.catalog.channels` and is empty when resolution is
unconstrained. The `catalog` label comes from the resolved catalog recorded in
status. A failed catalog refresh continues to expose the digest of the content
still being served.

Only `olm_cluster_catalog_serving{name,digest}` and the two unlabeled aggregate
series are forwarded through Telemeter. Detailed metric series remain
in-cluster; equivalent API fields are gathered through Insights.

For upgrade use cases, SREs correlate catalog digest changes with abnormal
counts; PM analysis uses Insights history. No separate update-event counter is
required.

### Risks and Mitigations

- Telemeter cardinality is limited to `C+2` active series per cluster, where
  `C` is the number of ClusterCatalogs. OCP provides three catalogs by default,
  but administrators can add more and digest changes create historical churn;
  representative clusters will be measured before allowlisting.
- Resolved catalog attribution could become inconsistent during upgrades; the
  status value remains tied to the bundle that is actually installed.

### Drawbacks

As OLM is a package manager, package, version, catalog, and channel values are
inherently unbounded; exporting them fleet-wide as metric labels would create
unbounded series. This design therefore trades fine-grained, real-time fleet
attribution for bounded cardinality. Telemeter can show that a catalog digest
change correlates with abnormal extensions, but package-level attribution
requires Insights or in-cluster investigation.

## Alternatives (Not Implemented)

- Forward detailed per-extension metrics through Telemeter: rejected because
  package, version, catalog, channel, and reason labels are unbounded.
- Infer the selected catalog from the `ClusterExtension` specification: rejected
  because a selector does not identify which catalog won resolution. Configured
  channel constraints are read directly from the specification.
- Add an update-event counter: rejected because the existing catalog digest,
  aggregate health, Insights history, and in-cluster metrics cover the agreed
  SRE and PM use cases.

## Open Questions [optional]

- Should resolved catalog be a scalar field on `status.install`, or part of a
  nested source structure?
- Do existing Insights/CCX reports consume the new archive data automatically, or
  is a separate backend/reporting work item required?

## Test Plan

- Unit and controller tests cover resolved catalog status across installs,
  upgrades, failed upgrades, and objects without provenance.
- Collector and recording-rule tests cover expected labels, missing status,
  deletion, cache failures, zero totals, and abnormal-state aggregation.
- Insights archive output and Telemeter allowlisting are validated with
  representative resources and cardinality measurements.

## Graduation Criteria

This proposal targets GA. It is complete when:

- The status API change is reviewed, documented, and shipped.
- The four in-cluster metrics and two recording rules are available by default
  with unit and end-to-end coverage.
- Insights archive collection and the three Telemeter series are verified.
- Representative cardinality measurements remain within the agreed budget.

## Upgrade / Downgrade Strategy

No user action is required. Existing ClusterExtensions remain valid but may lack
resolved catalog provenance until a successful install or upgrade records it.
Metrics appear when the updated components are deployed.

Older components ignore the additive status field. Downgrading removes the new
observability data but does not change installed extensions or catalog serving
behavior.

TODO: Document upgrade, downgrade, and telemetry behavior while OLM v0 and OLM v1
coexist during the migration period.

## Version Skew Strategy

The updated ClusterExtension CRD must be applied before the updated
operator-controller Deployment. CVO payload manifest ordering will place the
CRD before the controller within the existing OLM component/run level.

During rollout, metrics and Insights tolerate ClusterExtensions without the new
resolved catalog field. Recording rules and the Telemeter allowlist
also tolerate their source metrics being temporarily absent; data begins flowing
when the updated collectors are available.

There is no node or kubelet impact.

## Operational Aspects of API Extensions

Not applicable beyond adding a controller-populated field to an existing CRD
status. This introduces no webhook, aggregated API server, finalizer, or new
dependency affecting API availability or throughput.

## Support Procedures

No special support procedures are required. Failures are visible through the
existing OLM v1 resource status, operator logs, and monitoring tools. The
additive status field cannot be disabled independently; removing the CRD is
unsupported and would remove existing custom resources.

## Infrastructure Needed [optional]

No new infrastructure is required. The proposal uses the existing
operator-controller, catalogd, cluster monitoring, Insights, and Telemeter
infrastructure.
