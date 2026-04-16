FROM registry.ci.openshift.org/ocp/4.22:base-rhel9
USER 1001

LABEL io.k8s.display-name="OpenShift Operator Lifecycle Manager Operator Controller E2E Registry" \
  io.k8s.description="This image is no longer used. E2E tests now use per-scenario dynamic catalogs."
