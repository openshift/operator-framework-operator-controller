#!/usr/bin/env bash
set -euo pipefail

# Update go.mod replaces to the latest (or provided) OpenShift Kubernetes fork commit.
# Usage:
#   ./ocp-replace-exact.sh                # uses latest commit on OCP fork default branch
#   ./ocp-replace-exact.sh <commitSHA>    # uses specific commit

OCP_REPO="github.com/openshift/kubernetes"
OCP_REMOTE="https://github.com/openshift/kubernetes.git"

# OCP Ginkgo fork pin (as in your go.mod)
GINKGO_FORK="github.com/openshift/onsi-ginkgo/v2"
GINKGO_VERSION="v2.6.1-0.20250416174521-4eb003743b54"

# EXACT list you provided (all are staging modules EXCEPT the root "k8s.io/kubernetes").
STAGING_MODULES=(
  k8s.io/api
  k8s.io/apiextensions-apiserver
  k8s.io/apimachinery
  k8s.io/apiserver
  k8s.io/cli-runtime
  k8s.io/client-go
  k8s.io/cloud-provider
  k8s.io/cluster-bootstrap
  k8s.io/code-generator
  k8s.io/component-base
  k8s.io/component-helpers
  k8s.io/controller-manager
  k8s.io/cri-api
  k8s.io/cri-client
  k8s.io/csi-translation-lib
  k8s.io/dynamic-resource-allocation
  k8s.io/endpointslice
  k8s.io/externaljwt
  k8s.io/kube-aggregator
  k8s.io/kube-controller-manager
  k8s.io/kube-proxy
  k8s.io/kube-scheduler
  k8s.io/kubectl
  k8s.io/kubelet
  k8s.io/metrics
  k8s.io/mount-utils
  k8s.io/pod-security-admission
  k8s.io/sample-apiserver
  k8s.io/sample-cli-plugin
  k8s.io/sample-controller
)

die(){ echo "error: $*" >&2; exit 1; }
need(){ command -v "$1" >/dev/null 2>&1 || die "missing command: $1"; }

need go
need git
[[ -f go.mod ]] || die "go.mod not found; run this from your repository root"

# Accept a commit SHA or use latest on default branch.
OCP_COMMIT="${1:-}"
if [[ -z "$OCP_COMMIT" ]]; then
  echo "Discovering latest OCP commit from ${OCP_REMOTE}…"
  OCP_COMMIT="$(git ls-remote "$OCP_REMOTE" HEAD | awk '{print $1}')"
  [[ -n "$OCP_COMMIT" ]] || die "failed to discover latest commit from $OCP_REMOTE"
else
  echo "Using provided OCP commit: ${OCP_COMMIT}"
fi

# Resolve canonical pseudo-version (v0.0.0-YYYYMMDDHHMMSS-<sha>) for that commit.
# The module declares itself as k8s.io/kubernetes, so we can't use github.com/openshift/kubernetes
# directly. Instead, we'll construct the pseudo-version from git information.
export GOPROXY="https://proxy.golang.org,direct"

# The script uses GOPRIVATE and GONOSUMDB internally during version resolution, but the resulting go.mod and go.sum 
# files work without those environment variables, which is required for the downstreaming process with 
# operator-framework-tooling.
export GOPRIVATE="github.com/openshift/*"
export GONOSUMDB="github.com/openshift/*"

echo "Resolving pseudo-version for k8s.io/kubernetes@${OCP_COMMIT}…"

# Clone the repo temporarily to get commit timestamp
TMP_DIR=$(mktemp -d)
trap "rm -rf ${TMP_DIR}" EXIT

git clone --depth=1 "${OCP_REMOTE}" "${TMP_DIR}" >/dev/null 2>&1
cd "${TMP_DIR}"
git checkout "${OCP_COMMIT}" >/dev/null 2>&1 || {
    cd - >/dev/null
    die "could not checkout commit ${OCP_COMMIT}"
}

# Get commit timestamp in the format YYYYMMDDHHMMSS
COMMIT_TIME=$(git log -1 --format=%ct "${OCP_COMMIT}")
COMMIT_DATE=$(date -u -r "${COMMIT_TIME}" +%Y%m%d%H%M%S 2>/dev/null || date -u -d "@${COMMIT_TIME}" +%Y%m%d%H%M%S 2>/dev/null || echo "")
SHORT_SHA="${OCP_COMMIT:0:12}"

cd - >/dev/null
rm -rf "${TMP_DIR}"
trap - EXIT

if [[ -z "$COMMIT_DATE" ]]; then
    die "could not get commit timestamp for ${OCP_COMMIT}"
fi

# Construct pseudo-version: v0.0.0-YYYYMMDDHHMMSS-<short-sha>
OCP_VERSION="v0.0.0-${COMMIT_DATE}-${SHORT_SHA}"
echo "Resolved OCP version: ${OCP_VERSION}"

echo "Updating go.mod replaces…"

# Clean up any existing replace directives for k8s.io modules first
# This ensures we start with a clean slate
for m in k8s.io/kubernetes "${STAGING_MODULES[@]}"; do
    go mod edit -dropreplace "${m}" 2>/dev/null || true
done
go mod edit -dropreplace "github.com/onsi/ginkgo/v2" 2>/dev/null || true

# 1) OCP Ginkgo fork
go mod edit -replace "github.com/onsi/ginkgo/v2=${GINKGO_FORK}@${GINKGO_VERSION}"

# 2) Root k8s.io/kubernetes → OCP fork
go mod edit -replace "k8s.io/kubernetes=${OCP_REPO}@${OCP_VERSION}"

# 3) All staging modules → staging path in the OCP fork at the same version
for m in "${STAGING_MODULES[@]}"; do
  go mod edit -replace "${m}=${OCP_REPO}/staging/src/${m}@${OCP_VERSION}"
done

# 4) Tidy up
go mod tidy
go mod vendor

echo
echo "Done."
echo "  OCP commit:  ${OCP_COMMIT}"
echo "  OCP version: ${OCP_VERSION}"
echo "go.mod and go.sum vendor/ updated."
