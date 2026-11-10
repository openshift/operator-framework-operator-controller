#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

##################################################
# You shouldn't need to change anything below here
##################################################

# Know where the repo root is so we can reference things relative to it
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

DOWNSTREAM="${REPO_ROOT}/openshift"
HELMDIR="${DOWNSTREAM}/helm"
GO_TOOL="${DOWNSTREAM}/tools/run.sh"

# Pipe through yq to keep formatting.
"${GO_TOOL}" helm template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/operator-controller.yaml" | "${GO_TOOL}" yq > "${DOWNSTREAM}/operator-controller/manifests.yaml"
"${GO_TOOL}" helm template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/operator-controller.yaml" --values "${HELMDIR}/experimental.yaml" | "${GO_TOOL}" yq > "${DOWNSTREAM}/operator-controller/manifests-experimental.yaml"
"${GO_TOOL}" helm template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/catalogd.yaml" | "${GO_TOOL}" yq > "${DOWNSTREAM}/catalogd/manifests.yaml"
"${GO_TOOL}" helm template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/catalogd.yaml" --values "${HELMDIR}/experimental.yaml" | "${GO_TOOL}" yq > "${DOWNSTREAM}/catalogd/manifests-experimental.yaml"
"${GO_TOOL}" helm template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/object-controller.yaml" | "${GO_TOOL}" yq > "${DOWNSTREAM}/object-controller/manifests.yaml"
"${GO_TOOL}" helm template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/object-controller.yaml" --values "${HELMDIR}/experimental.yaml" | "${GO_TOOL}" yq > "${DOWNSTREAM}/object-controller/manifests-experimental.yaml"
