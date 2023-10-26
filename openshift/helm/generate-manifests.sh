#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

##################################################
# You shouldn't need to change anything below here
##################################################

# Know where the repo root is so we can reference things relative to it
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source bingo so we can use kustomize and yq
. "${REPO_ROOT}/openshift/.bingo/variables.env"

DOWNSTREAM="${REPO_ROOT}/openshift"
HELMDIR="${DOWNSTREAM}/helm"

# Pipe through ${YQ} to keep formatting
${HELM} template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/operator-controller.yaml" | ${YQ} > "${DOWNSTREAM}/operator-controller/manifests.yaml"
${HELM} template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/operator-controller.yaml" --values "${HELMDIR}/experimental.yaml" | ${YQ} > "${DOWNSTREAM}/operator-controller/manifests-experimental.yaml"
${HELM} template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/catalogd.yaml" | ${YQ} > "${DOWNSTREAM}/catalogd/manifests.yaml"
${HELM} template olmv1 "${REPO_ROOT}/helm/olmv1" --values "${HELMDIR}/catalogd.yaml" --values "${HELMDIR}/experimental.yaml" | ${YQ} > "${DOWNSTREAM}/catalogd/manifests-experimental.yaml"
