package helpers

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

const (
	// diagnosticMaxNames caps how many object names are printed per resource kind.
	diagnosticMaxNames = 25

	// diagnosticMaxEvents caps how many events are printed for a namespace.
	diagnosticMaxEvents = 25

	// diagnosticMaxMessage caps the length of a single condition or event message.
	diagnosticMaxMessage = 512
)

// diagnosticTimeout bounds the independent context used to capture failure-time
// diagnostics. Cleanup runs with a context that may already be expired or cancelled by
// the time a wait fails, so diagnostics must not inherit that cancellation - but they
// must not hang either. It is a var so tests can shorten it.
var diagnosticTimeout = 30 * time.Second

// DiagnosticReader is the read-only subset of client.Client that failure-time
// diagnostics need. Keeping the surface narrow lets unit tests drive the dump with a
// fake reader instead of a live cluster.
type DiagnosticReader interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
	List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error
}

// client.Client must stay usable as a DiagnosticReader: it is what the e2e cleanup
// paths pass in.
var _ DiagnosticReader = (client.Client)(nil)

// CleanupDiagnostics describes what to capture when a cleanup step fails.
//
// Everything it prints is an explicit, safe projection: object names, phases,
// finalizers, conditions and webhook wiring. It never prints Secret or ConfigMap
// contents, CA bundles, pod logs or whole objects.
type CleanupDiagnostics struct {
	// Namespace is the namespace being cleaned up. Its object state and a name-only
	// census of its contents are captured. Empty skips both.
	Namespace string

	// ClusterExtensionName is the cluster-scoped ClusterExtension to report on.
	// Empty skips the ClusterExtension section.
	ClusterExtensionName string

	// ValidatingWebhookPrefix and MutatingWebhookPrefix limit the webhook
	// configuration report to the configurations this test owns. An empty prefix
	// skips that kind.
	ValidatingWebhookPrefix string
	MutatingWebhookPrefix   string

	// DeleteErr is the error returned by the delete request that preceded the wait,
	// if any. It is reported so a failed delete is not lost behind the wait failure.
	DeleteErr error

	// IncludeEvents additionally captures a bounded, newest-first slice of the
	// namespace's events.
	IncludeEvents bool
}

// Run runs cleanup and, when cleanup does not complete normally, dumps the configured
// diagnostics before the original failure continues to propagate.
//
// Ginkgo and Gomega report an assertion failure by panicking, so a cleanup node that
// fails never reaches the statement after the assertion, and an AfterEach guarded by
// CurrentSpecReport().Failed() has already run by then. Unwinding through a deferred
// call is what makes failure-time capture actually execute on the failing path.
//
// On success nothing is read or printed, so a clean cleanup costs nothing.
func (d CleanupDiagnostics) Run(ctx context.Context, w io.Writer, r DiagnosticReader, cleanup func()) {
	runWithDiagnostics(w, cleanup, func() { d.Dump(ctx, w, r) })
}

// runWithDiagnostics calls dump when cleanup does not complete normally, whether it
// unwinds by panic (Gomega's failure mode) or by runtime.Goexit.
//
// dump is best effort: a panic inside it is recovered and reported so that diagnostics
// can never replace or mask the failure being unwound.
func runWithDiagnostics(w io.Writer, cleanup, dump func()) {
	completed := false
	defer func() {
		if completed {
			return
		}
		// Never let a diagnostic panic displace the failure being unwound, but do
		// say that it happened so a broken dump is not mistaken for an empty one.
		defer func() {
			if r := recover(); r != nil && w != nil {
				fmt.Fprintf(w, "[cleanup-diagnostics] dump panicked and was suppressed: %v\n", r)
			}
		}()
		dump()
	}()

	cleanup()
	completed = true
}

// Dump writes the configured diagnostics to w. Every step is best effort: read errors
// are reported inline and never abort the remaining sections.
func (d CleanupDiagnostics) Dump(ctx context.Context, w io.Writer, r DiagnosticReader) {
	if w == nil || r == nil {
		return
	}

	diagCtx, cancel := context.WithTimeout(detachContext(ctx), diagnosticTimeout)
	defer cancel()

	sectionHeaderTo(w, "[cleanup-diagnostics] namespace=%q clusterExtension=%q",
		d.Namespace, d.ClusterExtensionName)
	if d.DeleteErr != nil {
		fmt.Fprintf(w, "delete request error: %v\n", d.DeleteErr)
	}

	d.dumpNamespace(diagCtx, w, r)
	d.dumpClusterExtension(diagCtx, w, r)
	d.dumpNamespacedCensus(diagCtx, w, r)
	d.dumpWebhookConfigurations(diagCtx, w, r)
	if d.IncludeEvents {
		d.dumpEvents(diagCtx, w, r)
	}

	sectionHeaderTo(w, "[cleanup-diagnostics] end")
}

// detachContext keeps ctx's values but drops its cancellation and deadline, so an
// expired cleanup context cannot suppress capture.
func detachContext(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return context.WithoutCancel(ctx)
}

func (d CleanupDiagnostics) dumpNamespace(ctx context.Context, w io.Writer, r DiagnosticReader) {
	if d.Namespace == "" {
		return
	}
	subHeaderTo(w, "namespace %s", d.Namespace)

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: d.Namespace}, ns); err != nil {
		fmt.Fprintf(w, "%s\n", describeGetError(err))
		return
	}

	fmt.Fprintf(w, "phase: %s\n", ns.Status.Phase)
	fmt.Fprintf(w, "deletionTimestamp: %s\n", formatTimestamp(ns.DeletionTimestamp))
	fmt.Fprintf(w, "metadata.finalizers: %s\n", formatList(ns.Finalizers))
	fmt.Fprintf(w, "spec.finalizers: %s\n", formatList(ns.Spec.Finalizers))
	writeConditions(w, ns.Status.Conditions, func(c corev1.NamespaceCondition) (string, string, string, string) {
		return string(c.Type), string(c.Status), c.Reason, c.Message
	})
}

func (d CleanupDiagnostics) dumpClusterExtension(ctx context.Context, w io.Writer, r DiagnosticReader) {
	if d.ClusterExtensionName == "" {
		return
	}
	subHeaderTo(w, "clusterextension %s", d.ClusterExtensionName)

	ce := &olmv1.ClusterExtension{}
	if err := r.Get(ctx, client.ObjectKey{Name: d.ClusterExtensionName}, ce); err != nil {
		fmt.Fprintf(w, "%s\n", describeGetError(err))
		return
	}

	fmt.Fprintf(w, "deletionTimestamp: %s\n", formatTimestamp(ce.DeletionTimestamp))
	fmt.Fprintf(w, "metadata.finalizers: %s\n", formatList(ce.Finalizers))
	if ce.Status.Install != nil {
		fmt.Fprintf(w, "status.install.bundle: %s %s\n",
			ce.Status.Install.Bundle.Name, ce.Status.Install.Bundle.Version)
	} else {
		fmt.Fprintf(w, "status.install.bundle: <none>\n")
	}
	writeConditions(w, ce.Status.Conditions, func(c metav1.Condition) (string, string, string, string) {
		return c.Type, string(c.Status), c.Reason, c.Message
	})
}

// dumpNamespacedCensus prints the names of the objects still present in the namespace.
// Secrets and ConfigMaps are listed as metadata only, so their values are never read
// into this process, let alone printed.
func (d CleanupDiagnostics) dumpNamespacedCensus(ctx context.Context, w io.Writer, r DiagnosticReader) {
	if d.Namespace == "" {
		return
	}
	subHeaderTo(w, "remaining objects in namespace %s (names only)", d.Namespace)

	kinds := []struct {
		kind string
		list client.ObjectList
	}{
		{"pods", &corev1.PodList{}},
		{"services", &corev1.ServiceList{}},
		{"deployments", &appsv1.DeploymentList{}},
		{"replicasets", &appsv1.ReplicaSetList{}},
		{"statefulsets", &appsv1.StatefulSetList{}},
		{"daemonsets", &appsv1.DaemonSetList{}},
		{"serviceaccounts", &corev1.ServiceAccountList{}},
		{"persistentvolumeclaims", &corev1.PersistentVolumeClaimList{}},
		{"secrets", metadataOnlyList(corev1.SchemeGroupVersion.WithKind("SecretList"))},
		{"configmaps", metadataOnlyList(corev1.SchemeGroupVersion.WithKind("ConfigMapList"))},
	}

	for _, k := range kinds {
		label := k.kind + ":"
		if err := r.List(ctx, k.list, client.InNamespace(d.Namespace)); err != nil {
			fmt.Fprintf(w, "%-24s error: %v\n", label, err)
			continue
		}
		names, err := objectNames(k.list)
		if err != nil {
			fmt.Fprintf(w, "%-24s error: %v\n", label, err)
			continue
		}
		fmt.Fprintf(w, "%-24s %s\n", label, formatNames(names))
	}
}

// dumpWebhookConfigurations reports only the admission webhook configurations this test
// owns, identified by name prefix. CA bundles are deliberately not printed.
//
// The validating and mutating passes are spelled out separately rather than unified:
// the two list types share no interface over Webhooks, and the explicit form is shorter
// than the generics needed to merge them.
func (d CleanupDiagnostics) dumpWebhookConfigurations(ctx context.Context, w io.Writer, r DiagnosticReader) {
	if d.ValidatingWebhookPrefix == "" && d.MutatingWebhookPrefix == "" {
		return
	}
	subHeaderTo(w, "test-owned webhook configurations")

	if prefix := d.ValidatingWebhookPrefix; prefix != "" {
		const kind = "validatingwebhookconfiguration"
		list := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
		if err := r.List(ctx, list); err != nil {
			fmt.Fprintf(w, "%ss error: %v\n", kind, err)
		} else {
			matched := 0
			for i := range list.Items {
				cfg := &list.Items[i]
				if !strings.HasPrefix(cfg.Name, prefix) {
					continue
				}
				matched++
				writeWebhookConfigHeader(w, kind, cfg.Name, cfg.DeletionTimestamp)
				for j := range cfg.Webhooks {
					writeWebhookSummary(w, cfg.Webhooks[j].Name,
						cfg.Webhooks[j].FailurePolicy, cfg.Webhooks[j].ClientConfig.Service)
				}
			}
			writeNoneMatching(w, kind, prefix, matched)
		}
	}

	if prefix := d.MutatingWebhookPrefix; prefix != "" {
		const kind = "mutatingwebhookconfiguration"
		list := &admissionregistrationv1.MutatingWebhookConfigurationList{}
		if err := r.List(ctx, list); err != nil {
			fmt.Fprintf(w, "%ss error: %v\n", kind, err)
			return
		}
		matched := 0
		for i := range list.Items {
			cfg := &list.Items[i]
			if !strings.HasPrefix(cfg.Name, prefix) {
				continue
			}
			matched++
			writeWebhookConfigHeader(w, kind, cfg.Name, cfg.DeletionTimestamp)
			for j := range cfg.Webhooks {
				writeWebhookSummary(w, cfg.Webhooks[j].Name,
					cfg.Webhooks[j].FailurePolicy, cfg.Webhooks[j].ClientConfig.Service)
			}
		}
		writeNoneMatching(w, kind, prefix, matched)
	}
}

func writeWebhookConfigHeader(w io.Writer, kind, name string, deletionTimestamp *metav1.Time) {
	fmt.Fprintf(w, "%s %s (deletionTimestamp: %s)\n", kind, name, formatTimestamp(deletionTimestamp))
}

func writeNoneMatching(w io.Writer, kind, prefix string, matched int) {
	if matched == 0 {
		fmt.Fprintf(w, "%ss: <none matching %q>\n", kind, prefix)
	}
}

func (d CleanupDiagnostics) dumpEvents(ctx context.Context, w io.Writer, r DiagnosticReader) {
	if d.Namespace == "" {
		return
	}
	subHeaderTo(w, "newest events in namespace %s (max %d)", d.Namespace, diagnosticMaxEvents)

	events := &corev1.EventList{}
	if err := r.List(ctx, events, client.InNamespace(d.Namespace)); err != nil {
		fmt.Fprintf(w, "events error: %v\n", err)
		return
	}
	if len(events.Items) == 0 {
		fmt.Fprintf(w, "(no events)\n")
		return
	}

	items := slices.Clone(events.Items)
	slices.SortStableFunc(items, func(a, b corev1.Event) int {
		// Newest first.
		return eventTime(b).Compare(eventTime(a))
	})
	if len(items) > diagnosticMaxEvents {
		fmt.Fprintf(w, "(showing %d of %d events)\n", diagnosticMaxEvents, len(items))
		items = items[:diagnosticMaxEvents]
	}
	for i := range items {
		fmt.Fprintf(w, "%s %s/%s count=%d: %s\n",
			formatTime(eventTime(items[i])),
			items[i].Type, items[i].Reason, eventCount(items[i]),
			truncate(items[i].Message))
	}
}

func writeWebhookSummary(w io.Writer, name string,
	failurePolicy *admissionregistrationv1.FailurePolicyType,
	service *admissionregistrationv1.ServiceReference) {
	policy := "<unset>"
	if failurePolicy != nil {
		policy = string(*failurePolicy)
	}
	target := "<no service ref>"
	if service != nil {
		path := ""
		if service.Path != nil {
			path = *service.Path
		}
		target = fmt.Sprintf("%s/%s%s", service.Namespace, service.Name, path)
	}
	fmt.Fprintf(w, "  webhook %s failurePolicy=%s service=%s\n", name, policy, target)
}

// writeConditions prints a condition list. project maps one condition to its type,
// status, reason and message, so both corev1 and metav1 condition shapes are supported.
func writeConditions[T any](w io.Writer, conditions []T, project func(T) (string, string, string, string)) {
	if len(conditions) == 0 {
		fmt.Fprintf(w, "status.conditions: <none>\n")
		return
	}
	fmt.Fprintf(w, "status.conditions:\n")
	for _, c := range conditions {
		condType, status, reason, message := project(c)
		fmt.Fprintf(w, "  %s=%s reason=%s message=%s\n", condType, status, reason, truncate(message))
	}
}

// metadataOnlyList returns a list that makes the API server return object metadata
// only, so object payloads are never transferred.
func metadataOnlyList(listGVK schema.GroupVersionKind) client.ObjectList {
	list := &metav1.PartialObjectMetadataList{}
	list.SetGroupVersionKind(listGVK)
	return list
}

// objectNames returns the sorted names of the items in list.
func objectNames(list client.ObjectList) ([]string, error) {
	items, err := apimeta.ExtractList(list)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		accessor, err := apimeta.Accessor(item)
		if err != nil {
			return nil, err
		}
		names = append(names, accessor.GetName())
	}
	slices.Sort(names)
	return names, nil
}

func formatNames(names []string) string {
	if len(names) == 0 {
		return "<none>"
	}
	if len(names) > diagnosticMaxNames {
		return fmt.Sprintf("%s ... (%d of %d shown)",
			strings.Join(names[:diagnosticMaxNames], ", "), diagnosticMaxNames, len(names))
	}
	return strings.Join(names, ", ")
}

// formatList renders a slice of string-like values, such as finalizers.
func formatList[T ~string](values []T) string {
	if len(values) == 0 {
		return "<none>"
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, string(v))
	}
	return strings.Join(out, ", ")
}

func formatTimestamp(ts *metav1.Time) string {
	if ts == nil || ts.IsZero() {
		return "<none>"
	}
	return formatTime(ts.Time)
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "<none>"
	}
	return t.UTC().Format(time.RFC3339)
}

func describeGetError(err error) string {
	if apierrors.IsNotFound(err) {
		return "(not found)"
	}
	return fmt.Sprintf("error: %v", err)
}

// eventTime returns the most recent time known for e, preferring the series' last
// observed time used by the events.k8s.io API so that recurring events - typically the
// interesting ones - sort as recent rather than as first seen.
func eventTime(e corev1.Event) time.Time {
	if e.Series != nil && !e.Series.LastObservedTime.IsZero() {
		return e.Series.LastObservedTime.Time
	}
	if !e.LastTimestamp.IsZero() {
		return e.LastTimestamp.Time
	}
	if !e.EventTime.IsZero() {
		return e.EventTime.Time
	}
	return e.FirstTimestamp.Time
}

func eventCount(e corev1.Event) int32 {
	if e.Series != nil && e.Series.Count > 0 {
		return e.Series.Count
	}
	return e.Count
}

// truncate bounds a message and keeps the result valid UTF-8, because it ends up in the
// JUnit XML the test extension writes.
func truncate(s string) string {
	if len(s) <= diagnosticMaxMessage {
		return s
	}
	cut := s[:diagnosticMaxMessage]
	for len(cut) > 0 && !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut + "...(truncated)"
}
