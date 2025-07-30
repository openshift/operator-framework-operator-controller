package commons

import (
	"context"
	"encoding/json"
	"fmt"

	//nolint:staticcheck // ST1001: dot-imports for readability
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github/operator-framework-operator-controller/openshift/tests-extension/pkg/env"
)

const (
	// TypeInstalled represents the status condition indicating the resource has been successfully installed.
	TypeInstalled = "Installed"

	// TypeProgressing indicates that the resource is in the process of being reconciled or installed.
	TypeProgressing = "Progressing"
)

// FetchUnstructured retrieves an unstructured Kubernetes object by its group, version, kind, and name.
func FetchUnstructured(group, version, kind, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: version, Kind: kind})

	err := env.Get().K8sClient.Get(context.TODO(), client.ObjectKey{Name: name}, obj)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("failed to get %s %q", kind, name))
	return obj
}

// ExtractConditions retrieves the status conditions from an unstructured Kubernetes object.
func ExtractConditions(obj *unstructured.Unstructured) []metav1.Condition {
	raw, found, err := unstructured.NestedSlice(obj.Object, "status", "conditions")
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())

	data, err := json.Marshal(raw)
	Expect(err).NotTo(HaveOccurred())

	var conds []metav1.Condition
	err = json.Unmarshal(data, &conds)
	Expect(err).NotTo(HaveOccurred())
	return conds
}

// ExpectedConditionsMatch checks that the specified conditions exist on the resource
func ExpectedConditionsMatch(
	group, version, kind, name string,
	expected []metav1.Condition,
) func() {
	return func() {
		obj := FetchUnstructured(group, version, kind, name)
		actual := ExtractConditions(obj)

		for _, want := range expected {
			got := meta.FindStatusCondition(actual, want.Type)
			if got == nil || got.Status != want.Status {
				Expect(fmt.Sprintf("condition %q status = %v", want.Type, got)).To(Equal("expected match"),
					"Expected condition %q with status %q", want.Type, want.Status)
			}
		}
	}
}

// ExpectedConditionsWithMessages will check that the specified conditions exist on the resource
// and that their messages contain the expected substrings.
func ExpectedConditionsWithMessages(
	group, version, kind, name string,
	expected []metav1.Condition,
	messages map[string]string, // map[Type]MessageSubstring
) func() {
	return func() {
		obj := FetchUnstructured(group, version, kind, name)
		actual := ExtractConditions(obj)

		for _, want := range expected {
			got := meta.FindStatusCondition(actual, want.Type)
			Expect(got).ToNot(BeNil(), "expected condition %q to exist", want.Type)
			Expect(got.Status).To(Equal(want.Status), "unexpected status for condition %q", want.Type)

			if msgSubstr, ok := messages[want.Type]; ok && msgSubstr != "" {
				Expect(got.Message).To(ContainSubstring(msgSubstr), "unexpected message for condition %q", want.Type)
			}
		}
	}
}
