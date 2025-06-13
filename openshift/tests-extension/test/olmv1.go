package test

import (
	"context"

	//nolint:staticcheck // ST1001: dot-imports are acceptable here for readability in test code
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001: dot-imports are acceptable here for readability in test code
	. "github.com/onsi/gomega"
)

var _ = Describe("[sig-olmv1] OLMv1", func() {
	It("should pass a trivial sanity check", func(ctx context.Context) {
		Expect(len("test")).To(BeNumerically(">", 0))
	})
})
