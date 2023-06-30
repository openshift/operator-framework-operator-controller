FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.20-openshift-4.14 AS builder
WORKDIR /build
COPY . .
RUN make go-build-local

FROM registry.ci.openshift.org/ocp/4.14:base
USER 1001
COPY --from=builder /build/bin/manager /manager
LABEL io.k8s.display-name="OpenShift Operator Lifecycle Manager Operator Controller" \
  io.k8s.description="This is a component of OpenShift Container Platform that allows operator installation."
