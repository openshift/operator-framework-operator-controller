# Builds the test bin olmv1-tests-ext from the source code.
# The bin will be used in the final image (olm-operator-controller)
# so it can be used by openshift-tests.
# See: https://github.com/openshift/origin/blob/main/pkg/test/extensions/binary.go#L53-L62
FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.24-openshift-4.20 AS builder

WORKDIR /build
COPY . .

# Build the olmv1-tests-ext binary.
RUN make build && \
    gzip -f ./bin/olmv1-tests-ext

# Copy directly from the build output directory
FROM registry.ci.openshift.org/ocp/4.20:base-rhel9
COPY --from=builder /build/bin/olmv1-tests-ext.gz /usr/bin/olmv1-tests-ext.gz

LABEL io.k8s.display-name="OLMv1 Operator Controller" \
      io.openshift.release.operator=true \
      io.openshift.tags="openshift,tests,e2e,e2e-extension"
