# Builds the test bin olmv1-tests-ext from the source code.
# The bin will be used in the final image (olm-operator-controller)
# so it can be used by openshift-tests.
# See: https://github.com/openshift/origin/blob/main/pkg/test/extensions/binary.go#L53-L62
FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.24-openshift-4.20 AS builder

WORKDIR /go/src/github.com/operator-framework/operator-controller/openshift/tests-extension
COPY . .

RUN make build && \
    mkdir -p /tmp/build && \
    cp ./bin/olmv1-tests-ext /tmp/build/olmv1-tests-ext && \
    gzip -f /tmp/build/olmv1-tests-ext

FROM registry.ci.openshift.org/ocp/4.20:base-rhel9
COPY --from=builder /tmp/build/olmv1-tests-ext.gz /usr/bin/olmv1-tests-ext.gz

LABEL io.k8s.display-name="OLMv1 Operator Controller" \
      io.openshift.release.operator=true \
      io.openshift.tags="openshift,tests,e2e,e2e-extension"
