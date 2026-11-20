package helpers

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	olmv1 "github.com/operator-framework/operator-controller/api/v1"
)

const testNamespace = "webhook-operator-test"

// fakeReader is an in-package DiagnosticReader. The production interface is two
// methods wide on purpose, so the tests need no fake-client dependency and no cluster.
//
// It is deliberately strict: a Get or List the dump is not supposed to make is an
// error rather than an empty result. That is what turns "Secrets are never read in
// full" from a review-time claim into an assertion - see listErrUnexpected below.
type fakeReader struct {
	namespace        *corev1.Namespace
	clusterExtension *olmv1.ClusterExtension
	pods             []corev1.Pod
	services         []corev1.Service
	secretNames      []string
	configMapNames   []string
	events           []corev1.Event
	validating       []admissionregistrationv1.ValidatingWebhookConfiguration
	mutating         []admissionregistrationv1.MutatingWebhookConfiguration

	// err, when set, fails every call.
	err error
	// guard, when set, inspects the context of every call first.
	guard func(context.Context) error
}

var _ DiagnosticReader = (*fakeReader)(nil)

func (f *fakeReader) check(ctx context.Context) error {
	if f.guard != nil {
		if err := f.guard(ctx); err != nil {
			return err
		}
	}
	return f.err
}

func (f *fakeReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object,
	_ ...client.GetOption) error {
	if err := f.check(ctx); err != nil {
		return err
	}
	switch o := obj.(type) {
	case *corev1.Namespace:
		if f.namespace == nil {
			return apierrors.NewNotFound(corev1.Resource("namespaces"), key.Name)
		}
		*o = *f.namespace
	case *olmv1.ClusterExtension:
		if f.clusterExtension == nil {
			return apierrors.NewNotFound(
				schema.GroupResource{Group: olmv1.GroupVersion.Group, Resource: "clusterextensions"}, key.Name)
		}
		*o = *f.clusterExtension
	default:
		return fmt.Errorf("fakeReader: unexpected Get for %T", obj)
	}
	return nil
}

func (f *fakeReader) List(ctx context.Context, list client.ObjectList, _ ...client.ListOption) error {
	if err := f.check(ctx); err != nil {
		return err
	}
	switch l := list.(type) {
	case *corev1.PodList:
		l.Items = slices.Clone(f.pods)
	case *corev1.ServiceList:
		l.Items = slices.Clone(f.services)
	case *corev1.EventList:
		l.Items = slices.Clone(f.events)
	case *admissionregistrationv1.ValidatingWebhookConfigurationList:
		l.Items = slices.Clone(f.validating)
	case *admissionregistrationv1.MutatingWebhookConfigurationList:
		l.Items = slices.Clone(f.mutating)
	case *metav1.PartialObjectMetadataList:
		switch kind := l.GroupVersionKind().Kind; kind {
		case "SecretList":
			l.Items = partialMetadata(f.secretNames)
		case "ConfigMapList":
			l.Items = partialMetadata(f.configMapNames)
		default:
			return fmt.Errorf("fakeReader: unexpected metadata list kind %q", kind)
		}
	case *corev1.SecretList, *corev1.ConfigMapList:
		return fmt.Errorf("fakeReader: %T must be listed as metadata only", list)
	case *appsv1.DeploymentList, *appsv1.ReplicaSetList, *appsv1.StatefulSetList,
		*appsv1.DaemonSetList, *corev1.ServiceAccountList, *corev1.PersistentVolumeClaimList:
		// Intentionally left empty.
	default:
		return fmt.Errorf("fakeReader: unexpected List for %T", list)
	}
	return nil
}

func partialMetadata(names []string) []metav1.PartialObjectMetadata {
	items := make([]metav1.PartialObjectMetadata, 0, len(names))
	for _, n := range names {
		items = append(items, metav1.PartialObjectMetadata{
			ObjectMeta: metav1.ObjectMeta{Name: n, Namespace: testNamespace},
		})
	}
	return items
}

func terminatingNamespace() *corev1.Namespace {
	deletedAt := metav1.NewTime(time.Date(2026, 11, 10, 3, 20, 6, 0, time.UTC))
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:              testNamespace,
			DeletionTimestamp: &deletedAt,
			Finalizers:        []string{"example.com/holds-namespace"},
		},
		Spec: corev1.NamespaceSpec{
			Finalizers: []corev1.FinalizerName{corev1.FinalizerKubernetes},
		},
		Status: corev1.NamespaceStatus{
			Phase: corev1.NamespaceTerminating,
			Conditions: []corev1.NamespaceCondition{{
				Type:               corev1.NamespaceDeletionContentFailure,
				Status:             corev1.ConditionTrue,
				Reason:             "ContentDeletionFailed",
				LastTransitionTime: metav1.NewTime(time.Date(2026, 11, 10, 3, 19, 0, 0, time.UTC)),
				Message:            "failed calling webhook vwebhooktest-v1.kb.io",
			}},
		},
	}
}

func installedClusterExtension() *olmv1.ClusterExtension {
	return &olmv1.ClusterExtension{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testNamespace,
			Finalizers: []string{"olm.operatorframework.io/cleanup-contentmanager-cache"},
		},
		Status: olmv1.ClusterExtensionStatus{
			Install: &olmv1.ClusterExtensionInstallStatus{
				Bundle: olmv1.BundleMetadata{Name: "webhook-operator.v0.0.5", Version: "0.0.5"},
			},
			Conditions: []metav1.Condition{{
				Type:    olmv1.TypeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "Retrying",
				Message: "waiting for finalizer",
			}},
		},
	}
}

func testWebhookConfigurations(r *fakeReader) {
	failPolicy := admissionregistrationv1.Fail
	path := "/validate-webhook-operators-coreos-io-v1-webhooktest"
	r.validating = []admissionregistrationv1.ValidatingWebhookConfiguration{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "vwebhooktest-v1.kb.io"},
			Webhooks: []admissionregistrationv1.ValidatingWebhook{{
				Name:          "vwebhooktest-v1.kb.io",
				FailurePolicy: &failPolicy,
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("SUPER-SECRET-CA-BUNDLE"),
					Service: &admissionregistrationv1.ServiceReference{
						Namespace: testNamespace,
						Name:      "webhook-operator-controller-manager-service",
						Path:      &path,
					},
				},
			}},
		},
		{ObjectMeta: metav1.ObjectMeta{Name: "sre-unrelated-validation"}},
	}
	r.mutating = []admissionregistrationv1.MutatingWebhookConfiguration{
		// Deliberately has neither a failure policy nor a service reference.
		{
			ObjectMeta: metav1.ObjectMeta{Name: "mwebhooktest-v1.kb.io"},
			Webhooks:   []admissionregistrationv1.MutatingWebhook{{Name: "mwebhooktest-v1.kb.io"}},
		},
		{ObjectMeta: metav1.ObjectMeta{Name: "pod-identity-webhook"}},
	}
}

func fullDiagnostics() CleanupDiagnostics {
	return CleanupDiagnostics{
		Namespace:               testNamespace,
		ClusterExtensionName:    testNamespace,
		ValidatingWebhookPrefix: "vwebhooktest",
		MutatingWebhookPrefix:   "mwebhooktest",
		IncludeEvents:           true,
	}
}

func dumpToString(d CleanupDiagnostics, ctx context.Context, r DiagnosticReader) string { //nolint:revive // ctx after the receiver-like arg keeps call sites readable
	var sb strings.Builder
	d.Dump(ctx, &sb, r)
	return sb.String()
}

func assertContains(t *testing.T, out string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(out, w) {
			t.Errorf("diagnostics output missing %q\n--- output ---\n%s", w, out)
		}
	}
}

func assertNotContains(t *testing.T, out string, forbidden ...string) {
	t.Helper()
	for _, f := range forbidden {
		if strings.Contains(out, f) {
			t.Errorf("diagnostics output must not contain %q\n--- output ---\n%s", f, out)
		}
	}
}

// TestDumpCapturesCleanupState covers the state a stalled namespace cleanup needs:
// namespace phase/deletionTimestamp/finalizers/conditions, ClusterExtension status, a
// name-only census and only the test's own webhook configurations.
func TestDumpCapturesCleanupState(t *testing.T) {
	r := &fakeReader{
		namespace:        terminatingNamespace(),
		clusterExtension: installedClusterExtension(),
		pods: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{
			Name: "webhook-operator-controller-manager-abc", Namespace: testNamespace}}},
		services: []corev1.Service{{ObjectMeta: metav1.ObjectMeta{
			Name: "webhook-operator-controller-manager-service", Namespace: testNamespace}}},
		secretNames:    []string{"webhook-service-cert"},
		configMapNames: []string{"kube-root-ca.crt"},
		events: []corev1.Event{{
			ObjectMeta:    metav1.ObjectMeta{Name: "evt-1", Namespace: testNamespace},
			Type:          corev1.EventTypeWarning,
			Reason:        "FailedDelete",
			Count:         3,
			LastTimestamp: metav1.NewTime(time.Date(2026, 11, 10, 3, 21, 0, 0, time.UTC)),
		}},
	}
	testWebhookConfigurations(r)

	out := dumpToString(fullDiagnostics(), context.Background(), r)

	assertContains(t, out,
		"[cleanup-diagnostics]",
		"phase: Terminating",
		"deletionTimestamp: 2026-11-10T03:20:06Z",
		"metadata.finalizers: example.com/holds-namespace",
		"spec.finalizers: kubernetes",
		"NamespaceDeletionContentFailure=True",
		"reason=ContentDeletionFailed",
		"lastTransitionTime=2026-11-10T03:19:00Z",
		"status.install.bundle: webhook-operator.v0.0.5 0.0.5",
		"olm.operatorframework.io/cleanup-contentmanager-cache",
		"Progressing=True",
		"webhook-operator-controller-manager-abc",
		"webhook-service-cert",
		"kube-root-ca.crt",
		"validatingwebhookconfiguration vwebhooktest-v1.kb.io",
		"failurePolicy=Fail",
		fmt.Sprintf("service=%s/webhook-operator-controller-manager-service", testNamespace),
		"mutatingwebhookconfiguration mwebhooktest-v1.kb.io",
		"failurePolicy=<unset>",
		"service=<no service ref>",
		"count=3",
		"FailedDelete",
		"[cleanup-diagnostics] end",
	)

	// Safe projections only: no CA bundles, no webhook configurations belonging to
	// other components, no oversized messages. Secret and ConfigMap values cannot
	// appear at all, because fakeReader rejects any attempt to list them in full.
	assertNotContains(t, out,
		"SUPER-SECRET-CA-BUNDLE",
		"sre-unrelated-validation",
		"pod-identity-webhook",
		"must be listed as metadata only",
	)
}

// TestDumpReportsDeleteErrorAndEmptyState covers the delete-request failure path and
// the "nothing left to report" branches.
func TestDumpReportsDeleteErrorAndEmptyState(t *testing.T) {
	d := fullDiagnostics()
	d.DeleteErr = apierrors.NewInternalError(errors.New("admission webhook denied the delete"))

	out := dumpToString(d, context.Background(), &fakeReader{})

	assertNotContains(t, out, "admission webhook denied the delete")
	assertContains(t, out,
		"delete request error: InternalError(code=500)",
		"(not found)",
		`validatingwebhookconfigurations: <none matching "vwebhooktest">`,
		`mutatingwebhookconfigurations: <none matching "mwebhooktest">`,
		"pods:",
		"<none>",
		"(no events)",
		"[cleanup-diagnostics] end",
	)
}

// TestDumpReportsEmptyCollections covers the "<none>" rendering for objects that exist
// but carry no finalizers or conditions.
func TestDumpReportsEmptyCollections(t *testing.T) {
	out := dumpToString(fullDiagnostics(), context.Background(), &fakeReader{
		namespace:        &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}},
		clusterExtension: &olmv1.ClusterExtension{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}},
	})

	assertContains(t, out,
		"deletionTimestamp: <none>",
		"metadata.finalizers: <none>",
		"spec.finalizers: <none>",
		"status.conditions: <none>",
		"status.install.bundle: <none>",
	)
}

// TestDumpUsesIndependentContext proves that a cleanup context which has already been
// cancelled cannot suppress capture, while context values are still carried through.
func TestDumpUsesIndependentContext(t *testing.T) {
	type ctxKey struct{}

	r := &fakeReader{
		namespace: terminatingNamespace(),
		guard: func(ctx context.Context) error {
			if err := ctx.Err(); err != nil {
				return fmt.Errorf("request rejected: %w", err)
			}
			if ctx.Value(ctxKey{}) != "carried" {
				return errors.New("context value lost")
			}
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey{}, "carried"))
	cancel() // the cleanup context is already dead when the wait fails

	out := dumpToString(fullDiagnostics(), ctx, r)

	assertNotContains(t, out, "request rejected", "context value lost")
	assertContains(t, out, "phase: Terminating")
}

// TestDumpIsBoundedByDiagnosticTimeout proves the detached context is still bounded, so
// an unresponsive API server cannot wedge cleanup.
func TestDumpIsBoundedByDiagnosticTimeout(t *testing.T) {
	original := diagnosticTimeout
	diagnosticTimeout = 50 * time.Millisecond
	defer func() { diagnosticTimeout = original }()

	r := &fakeReader{guard: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}}

	done := make(chan string, 1)
	go func() { done <- dumpToString(fullDiagnostics(), context.Background(), r) }()

	select {
	case out := <-done:
		assertContains(t, out, "DeadlineExceeded", "[cleanup-diagnostics] end")
	case <-time.After(30 * time.Second):
		t.Fatal("Dump did not honour the diagnostic timeout")
	}
}

// TestDumpIsBestEffortOnReadErrors proves that read failures are reported but never
// abort the remaining sections.
func TestDumpIsBestEffortOnReadErrors(t *testing.T) {
	boom := errors.New("apiserver unavailable")

	out := dumpToString(fullDiagnostics(), context.Background(), &fakeReader{err: boom})

	assertNotContains(t, out, boom.Error())
	assertContains(t, out, "Unclassified(type=", "[cleanup-diagnostics] end")
}

// TestDumpBoundsCensusAndEvents proves the census and event output stay bounded and
// that recurring (series) events are ranked by their last observed time.
func TestDumpBoundsCensusAndEvents(t *testing.T) {
	r := &fakeReader{namespace: terminatingNamespace()}
	for i := range diagnosticMaxNames + 5 {
		r.secretNames = append(r.secretNames, fmt.Sprintf("secret-%03d", i))
	}
	for i := range diagnosticMaxEvents + 5 {
		r.events = append(r.events, corev1.Event{
			ObjectMeta:    metav1.ObjectMeta{Name: fmt.Sprintf("evt-%03d", i), Namespace: testNamespace},
			Reason:        fmt.Sprintf("Reason%03d", i),
			LastTimestamp: metav1.NewTime(time.Date(2026, 11, 10, 3, 20, i, 0, time.UTC)),
		})
	}
	// A recurring event whose first occurrence is the oldest of all, but which is
	// still firing: it must survive the cap and report the series count.
	r.events = append(r.events, corev1.Event{
		ObjectMeta: metav1.ObjectMeta{Name: "evt-recurring", Namespace: testNamespace},
		Reason:     "RecurringRootCause",
		EventTime:  metav1.NewMicroTime(time.Date(2026, 11, 10, 3, 0, 0, 0, time.UTC)),
		Series: &corev1.EventSeries{
			Count:            931,
			LastObservedTime: metav1.NewMicroTime(time.Date(2026, 11, 10, 4, 0, 0, 0, time.UTC)),
		},
	})

	out := dumpToString(fullDiagnostics(), context.Background(), r)

	assertContains(t, out,
		fmt.Sprintf("(%d of %d shown)", diagnosticMaxNames, diagnosticMaxNames+5),
		fmt.Sprintf("(showing %d of %d events)", diagnosticMaxEvents, diagnosticMaxEvents+6),
		"RecurringRootCause",
		"count=931",
		fmt.Sprintf("Reason%03d", diagnosticMaxEvents+4),
	)
	// Oldest one-shot events are dropped, and the recurring one is ranked first.
	assertNotContains(t, out, "Reason000")
	if i := strings.Index(out, "RecurringRootCause"); i < 0 || i > strings.Index(out, "Reason029") {
		t.Errorf("recurring event not ranked newest-first\n--- output ---\n%s", out)
	}
}

// TestDumpSkipsDisabledSections keeps the dump off resources the caller did not ask for.
func TestDumpSkipsDisabledSections(t *testing.T) {
	out := dumpToString(CleanupDiagnostics{Namespace: testNamespace}, context.Background(),
		&fakeReader{namespace: terminatingNamespace(), clusterExtension: installedClusterExtension()})

	assertNotContains(t, out, "clusterextension ", "webhook configurations", "newest events")
	assertContains(t, out, "phase: Terminating")
}

func TestDumpIgnoresNilWriterReaderOrContext(t *testing.T) {
	var sb strings.Builder

	// Must not panic.
	fullDiagnostics().Dump(context.Background(), nil, &fakeReader{})
	fullDiagnostics().Dump(context.Background(), &sb, nil)
	if sb.Len() != 0 {
		t.Errorf("expected no output with a nil reader, got %q", sb.String())
	}

	//nolint:staticcheck // SA1012: a nil context must not break failure-time capture.
	fullDiagnostics().Dump(nil, &sb, &fakeReader{namespace: terminatingNamespace()})
	assertContains(t, sb.String(), "phase: Terminating")
}

// TestRunSkipsDumpOnSuccess keeps a clean cleanup free of diagnostic cost.
func TestRunSkipsDumpOnSuccess(t *testing.T) {
	var sb strings.Builder
	fullDiagnostics().Run(context.Background(), &sb,
		&fakeReader{namespace: terminatingNamespace()}, func() {})
	if sb.Len() != 0 {
		t.Errorf("diagnostics ran for a cleanup that succeeded: %q", sb.String())
	}
}

// TestRunDumpsOnFailure wires the real dump to the real failure path.
func TestRunDumpsOnFailure(t *testing.T) {
	failure := errors.New("Timed out after 300.002s: namespace still exists")
	var sb strings.Builder

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		fullDiagnostics().Run(context.Background(), &sb,
			&fakeReader{namespace: terminatingNamespace()}, func() { panic(failure) })
	}()

	err, ok := recovered.(error)
	if !ok || !errors.Is(err, failure) {
		t.Fatalf("original failure not preserved, got %v", recovered)
	}
	assertContains(t, sb.String(), "phase: Terminating", "[cleanup-diagnostics] end")
}

// TestRunWithDiagnosticsPreservesPanic covers the Gomega failure mode, which unwinds by
// panicking.
func TestRunWithDiagnosticsPreservesPanic(t *testing.T) {
	failure := errors.New("primary failure")
	dumped := false

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		runWithDiagnostics(nil, func() { panic(failure) }, func() { dumped = true })
	}()

	if !dumped {
		t.Error("diagnostics did not run for a failing cleanup")
	}
	err, ok := recovered.(error)
	if !ok || !errors.Is(err, failure) {
		t.Errorf("original failure not preserved, got %v", recovered)
	}
}

// TestRunWithDiagnosticsPreservesGoexit pins the other unwind mode Ginkgo may use, so a
// future refactor to a recover()-based implementation cannot silently skip the dump.
func TestRunWithDiagnosticsPreservesGoexit(t *testing.T) {
	dumped, reachedAfter := false, false
	done := make(chan struct{})

	go func() {
		defer close(done)
		runWithDiagnostics(nil, runtime.Goexit, func() { dumped = true })
		reachedAfter = true
	}()
	<-done

	if !dumped {
		t.Error("diagnostics did not run while unwinding a Goexit")
	}
	if reachedAfter {
		t.Error("Goexit was swallowed")
	}
}

// TestRunWithDiagnosticsReportsButDoesNotPropagateDumpPanic asserts a broken dump can
// neither replace the failure under investigation nor vanish silently.
func TestRunWithDiagnosticsReportsButDoesNotPropagateDumpPanic(t *testing.T) {
	failure := errors.New("primary failure")
	var sb strings.Builder

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		runWithDiagnostics(&sb,
			func() { panic(failure) },
			func() { panic(errors.New("diagnostics exploded")) },
		)
	}()

	err, ok := recovered.(error)
	if !ok || !errors.Is(err, failure) {
		t.Errorf("diagnostic panic masked the original failure, got %v", recovered)
	}
	assertNotContains(t, sb.String(), "diagnostics exploded")
	assertContains(t, sb.String(),
		"dump panicked and was suppressed", "recovered value of type")
}

func TestMetadataOnlyListRequestsMetadataOnly(t *testing.T) {
	gvk := corev1.SchemeGroupVersion.WithKind("SecretList")

	pm, ok := metadataOnlyList(gvk).(*metav1.PartialObjectMetadataList)
	if !ok {
		t.Fatalf("expected a PartialObjectMetadataList, got %T", metadataOnlyList(gvk))
	}
	if got := pm.GroupVersionKind(); got != gvk {
		t.Errorf("unexpected GVK %v, want %v", got, gvk)
	}
}

// sentinels stand in for the kinds of content that free-form Kubernetes messages are
// known to carry: cluster-internal endpoints, credentials, addresses and URLs. None of
// them may reach the diagnostics output, which is published as a CI job artifact.
var sentinels = []string{
	"etcd-0.internal.cluster.example.corp",
	"sha256~AbCdEf0123456789SecretToken",
	"cluster-admin@internal.example.corp",
	"https://10.0.22.155:6443/healthz",
}

func sentinelText(prefix string) string {
	return prefix + ": " + strings.Join(sentinels, " ")
}

// TestDumpOmitsFreeFormText injects sentinel content into every free-form string the
// dump can reach - namespace and ClusterExtension condition messages, event messages,
// and the error returned by the reader - and asserts none of it is printed while the
// structured context stays useful.
func TestDumpOmitsFreeFormText(t *testing.T) {
	ns := terminatingNamespace()
	ns.Status.Conditions[0].Message = sentinelText("namespace condition")

	ce := installedClusterExtension()
	ce.Status.Conditions[0].Message = sentinelText("clusterextension condition")

	r := &fakeReader{
		namespace:        ns,
		clusterExtension: ce,
		events: []corev1.Event{{
			ObjectMeta:    metav1.ObjectMeta{Name: "evt-leak", Namespace: testNamespace},
			Type:          corev1.EventTypeWarning,
			Reason:        "FailedDelete",
			Count:         7,
			Message:       sentinelText("event"),
			LastTimestamp: metav1.NewTime(time.Date(2026, 11, 10, 3, 21, 0, 0, time.UTC)),
		}},
	}

	d := fullDiagnostics()
	d.DeleteErr = apierrors.NewForbidden(
		schema.GroupResource{Resource: "namespaces"}, testNamespace, errors.New(sentinelText("delete")))

	out := dumpToString(d, context.Background(), r)

	assertNotContains(t, out, sentinels...)
	// The structured context that makes the dump worth printing survives.
	assertContains(t, out,
		"phase: Terminating",
		"reason=ContentDeletionFailed",
		"reason=Retrying",
		"Warning/FailedDelete",
		"count=7",
		"delete request error: Forbidden(code=403)",
		"metadata.finalizers: example.com/holds-namespace",
	)
}

// TestDumpOmitsFreeFormReaderErrors covers the reader error paths, which reach every
// section of the dump.
func TestDumpOmitsFreeFormReaderErrors(t *testing.T) {
	out := dumpToString(fullDiagnostics(), context.Background(),
		&fakeReader{err: errors.New(sentinelText("reader"))})

	assertNotContains(t, out, sentinels...)
	assertContains(t, out, "Unclassified(type=", "[cleanup-diagnostics] end")
}

// TestRunWithDiagnosticsOmitsFreeFormPanicValue covers the recovered panic value, which
// is arbitrary data chosen by whatever panicked.
func TestRunWithDiagnosticsOmitsFreeFormPanicValue(t *testing.T) {
	failure := errors.New("primary failure")
	var sb strings.Builder

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		runWithDiagnostics(&sb,
			func() { panic(failure) },
			func() { panic(sentinelText("panic")) },
		)
	}()

	err, ok := recovered.(error)
	if !ok || !errors.Is(err, failure) {
		t.Errorf("diagnostic panic masked the original failure, got %v", recovered)
	}
	assertNotContains(t, sb.String(), sentinels...)
	assertContains(t, sb.String(), "dump panicked and was suppressed", "recovered value of type string")
}

func TestDescribeAPIError(t *testing.T) {
	secret := errors.New(sentinelText("api"))
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, "<none>"},
		{"deadline", fmt.Errorf("get: %w", context.DeadlineExceeded), "DeadlineExceeded"},
		{"cancelled", fmt.Errorf("list: %w", context.Canceled), "Canceled"},
		{"notFound", apierrors.NewNotFound(corev1.Resource("namespaces"), "ns"), "NotFound(code=404)"},
		{"forbidden", apierrors.NewForbidden(corev1.Resource("namespaces"), "ns", secret), "Forbidden(code=403)"},
		{"internal", apierrors.NewInternalError(secret), "InternalError(code=500)"},
		{"timeout", apierrors.NewTimeoutError(sentinelText("timeout"), 1), "Timeout(code=504)"},
		{"unrecognisedReason", &apierrors.StatusError{ErrStatus: metav1.Status{
			Reason: metav1.StatusReason(sentinelText("reason")), Code: 599,
		}}, "Unrecognized(code=599)"},
		{"noReason", &apierrors.StatusError{ErrStatus: metav1.Status{Code: 418}}, "NoReason(code=418)"},
		{"plain", secret, "Unclassified(type=*errors.errorString)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := DescribeAPIError(tc.err)
			if got != tc.want {
				t.Errorf("DescribeAPIError = %q, want %q", got, tc.want)
			}
			for _, s := range sentinels {
				if strings.Contains(got, s) {
					t.Errorf("DescribeAPIError leaked %q in %q", s, got)
				}
			}
		})
	}
}
