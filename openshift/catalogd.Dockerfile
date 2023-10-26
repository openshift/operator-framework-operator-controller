FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.23-openshift-4.19 AS builder
WORKDIR /build
COPY . .
RUN make -C catalogd go-build-local

FROM registry.ci.openshift.org/ocp/4.19:base-rhel9
USER 1001
COPY --from=builder /build/catalogd/bin/catalogd /catalogd
COPY openshift/catalogd/manifests /openshift/manifests

LABEL io.k8s.display-name="OpenShift Operator Lifecycle Manager Catalog Controller" \
      io.k8s.description="This is a component of OpenShift Container Platform that provides operator catalog support."
