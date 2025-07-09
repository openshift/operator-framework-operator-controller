FROM registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.24-openshift-4.20 AS builder
WORKDIR /build
COPY . .
# TODO Modify upstream Makefile to separate the 'go build' commands
# from 'image-registry' target so we don't need these
RUN go build -o ./push     ./testdata/push/push.go

FROM registry.ci.openshift.org/ocp/4.20:base-rhel9
USER 1001
COPY --from=builder /build/push /push
COPY openshift/operator-controller/manifests /openshift/manifests
COPY testdata/images /images

LABEL io.k8s.display-name="OpenShift Operator Lifecycle Manager Operator Controller E2E Registry" \
  io.k8s.description="This is a registry image that is used during E2E testing of Operator Controller"
