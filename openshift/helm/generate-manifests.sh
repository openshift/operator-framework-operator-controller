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

# This function generates the manifests
generate () {
    values=()
    OUTPUT_DIR=""
    local OPTIND
    while getopts "v:o:n:c:i:" arg; do
          case ${arg} in
              v)
                  values+=(${OPTARG})
                  ;;
              o)
                  OUTPUT_DIR=${OPTARG}
                  ;;
              *)
                  echo "bad argument ${arg}"
                  exit 1
                  ;;
          esac
    done

    DOWNSTREAM="${REPO_ROOT}/openshift/helm"

    VALUES=""
    for i in "${values[@]}"; do
        VALUES="${VALUES} --values ${DOWNSTREAM}/${i}"
    done

    # We're going to do file manipulation, so let's work in a temp dir
    TMP_ROOT="$(mktemp -p . -d 2>/dev/null || mktemp -d ./tmpdir.XXXXXXX)"
    # Make sure to delete the temp dir when we exit
    trap 'rm -rf $TMP_ROOT' EXIT

    # Copy upstream chart to temp
    cp -a "${REPO_ROOT}/helm" "${TMP_ROOT}/helm"

    # Create a temp dir for manifests
    TMP_MANIFEST_DIR="${TMP_ROOT}/manifests"
    mkdir -p "$TMP_MANIFEST_DIR"

    # Run helm template, which emits a single yaml file
    TMP_HELM_OUTPUT="${TMP_MANIFEST_DIR}/temp.yaml"
    ${HELM} template olmv1 "${TMP_ROOT}/helm/olmv1" ${VALUES} > "${TMP_HELM_OUTPUT}"

    # Use yq to split the single yaml file into 1 per document.
    # Naming convention: $index-$kind-$namespace-$name. If $namespace is empty, just use the empty string.
    (
        cd "$TMP_MANIFEST_DIR"

        # shellcheck disable=SC2016
        ${YQ} -s '$index +"-"+ (.kind|downcase) +"-"+ (.metadata.namespace // "") +"-"+ .metadata.name' temp.yaml
    )

    # Delete the single yaml file
    rm "${TMP_HELM_OUTPUT}"

    # Delete and recreate the actual manifests directory
    MANIFEST_DIR="${REPO_ROOT}/openshift/${OUTPUT_DIR}"
    rm -rf "${MANIFEST_DIR}"
    mkdir -p "${MANIFEST_DIR}"

    # Copy everything we just generated and split into the actual manifests directory
    cp "$TMP_MANIFEST_DIR"/* "$MANIFEST_DIR"/

    # Update file names to be in the format nn-$kind-$namespace-$name
    (
        cd "$MANIFEST_DIR"

        for f in *; do
            # Get the numeric prefix from the filename
            index=$(echo "$f" | cut -d '-' -f 1)
            # Keep track of the full file name without the leading number and dash
            name_without_index=${f#$index-}
            # Fix the double dash in cluster-scoped names
            name_without_index=${name_without_index//--/-}
            # Reformat the name so the leading number is always padded to 2 digits
            new_name=$(printf "%02d" "$index")-$name_without_index
            # Some file names (namely CRDs) don't end in .yml - make them
            if ! [[ "$new_name" =~ yml$ ]]; then
                new_name="${new_name}".yml
            fi
            if [[ "$f" != "$new_name" ]]; then
                # Rename
                mv "$f" "${new_name}"
            fi
        done
    )
    rm -rf "$TMP_ROOT"
}

#Generate the manifests
generate -v operator-controller.yaml -o operator-controller/manifests
generate -v operator-controller.yaml -v experimental.yaml -o operator-controller/manifests-experimental
generate -v catalogd.yaml -o catalogd/manifests
generate -v catalogd.yaml -v experimental.yaml -o catalogd/manifests-experimental
