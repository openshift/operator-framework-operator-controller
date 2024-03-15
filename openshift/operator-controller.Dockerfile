FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.21-openshift-4.16 AS builder
WORKDIR /build
COPY . .
RUN make go-build-local
RUN cd openshift && make build

FROM registry.ci.openshift.org/ocp/4.16:base
USER 1001
COPY --from=builder /build/bin/manager /manager
COPY --from=builder /build/openshift/bin/webhook /webhook
COPY openshift/manifests /openshift/manifests

LABEL io.k8s.display-name="OpenShift Operator Lifecycle Manager Operator Controller" \
  io.k8s.description="This is a component of OpenShift Container Platform that allows operator installation."
