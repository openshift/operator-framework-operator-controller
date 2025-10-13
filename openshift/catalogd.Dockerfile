FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.24-openshift-4.21 AS builder

ARG SOURCE_GIT_COMMIT
ENV GIT_COMMIT=${SOURCE_GIT_COMMIT}
WORKDIR /build
COPY . .
RUN make go-build-local

FROM registry.ci.openshift.org/ocp/4.21:base-rhel9
USER 1001
COPY --from=builder /build/bin/catalogd /catalogd
COPY openshift/catalogd/cp-manifests /cp-manifests
COPY helm/olmv1 /openshift/helm/olmv1
COPY openshift/helm/experimental.yaml /openshift/helm
COPY openshift/helm/catalogd.yaml /openshift/helm/openshift.yaml

LABEL io.k8s.display-name="OpenShift Operator Lifecycle Manager Catalog Controller" \
      io.k8s.description="This is a component of OpenShift Container Platform that provides operator catalog support."
