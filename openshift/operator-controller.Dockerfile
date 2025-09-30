FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.24-openshift-4.21 AS builder

ARG SOURCE_GIT_COMMIT
ENV GIT_COMMIT=${SOURCE_GIT_COMMIT}
WORKDIR /build
COPY . .
RUN make -f openshift/Makefile go-build-local && \
    # Build the OLMv1 Test Extension binary.
    # This is used by openshift/origin to allow us to register the OLMv1 test extension
    # The binary needs to be added in the component image and OCP image
    cd openshift/tests-extension && \
       make build && \
       mkdir -p /tmp/build && \
       cp ./bin/olmv1-tests-ext /tmp/build/olmv1-tests-ext && \
       gzip -f /tmp/build/olmv1-tests-ext

FROM registry.ci.openshift.org/ocp/4.21:base-rhel9
USER 1001
COPY --from=builder /build/bin/operator-controller /operator-controller
COPY --from=builder /tmp/build/olmv1-tests-ext.gz /usr/bin/olmv1-tests-ext.gz
COPY openshift/operator-controller/cp-manifests /cp-manifests
COPY openshift/operator-controller/manifests /openshift/manifests
COPY openshift/operator-controller/manifests-experimental /openshift/manifests-experimental
COPY helm/olmv1 /openshift/helm/olmv1
COPY openshift/helm/experimental.yaml /openshift/helm
COPY openshift/helm/operator-controller.yaml /openshift/helm/openshift.yaml

LABEL io.k8s.display-name="OpenShift Operator Lifecycle Manager Operator Controller" \
  io.k8s.description="This is a component of OpenShift Container Platform that allows operator installation."
