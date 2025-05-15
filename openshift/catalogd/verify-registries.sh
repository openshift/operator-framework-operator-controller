#!/usr/bin/env bash

# error=0, info=1, debug=2
LOGGING=${LOGGING:-1}

# These packages are used in the openshift/origin tests
# There is no automated way to get them, so this will have to suffice for now
packages=("elasticsearch-operator.v5.8.13" "openshift-pipelines-operator-rh.v1.18.0" "quay-operator.v3.13.0")

# tools that are needed for this script
YQ=${YQ:-yq}
OPM=${OPM:-opm}

prereq=(${OPM} ${YQ})

TMP_ROOT="$(mktemp -d)"
# Make sure to delete the temp dir when we exit
trap 'chmod -R +w ${TMP_ROOT} && rm -rf ${TMP_ROOT}' EXIT

debug() {
    if [ ${LOGGING} -ge 2 ]; then
        echo "DEBUG: $1"
    fi
}

info() {
    if [ ${LOGGING} -ge 1 ]; then
        echo "INFO: $1"
    fi
}

error() {
    if [ ${LOGGING} -ge 0 ]; then
        echo "ERROR: $1"
    fi
}

check_prereq() {
    if ! command -v $1 &> /dev/null; then
        error "unable to find prerequisite: $1"
        exit 1
    fi
}

for p in ${prereq[@]}; do
    check_prereq ${p}
done

catalogs=$(ls catalogd/manifests/*-clustercatalog-*)

for c in ${catalogs}; do
    reg=$(${YQ} .spec.source.image.ref ${c})
    out=${TMP_ROOT}/$(echo ${reg} | tr '/:.' '-').json
    debug "rendering ${reg} to ${out}"
    ${OPM} render ${reg} > ${out}
done

rendered=$(ls ${TMP_ROOT})

retcode=0

for p in ${packages[@]}; do
    debug "searching for ${p}"
    found='false'
    for r in ${rendered}; do
        result=$(jq -cs ".[] | select( .schema == \"olm.bundle\" ) | select( .name == \"${p}\" ) | {\"name\": .name, \"image\": .image}" ${TMP_ROOT}/${r})
        if [ -n "${result}" ]; then
            debug "${result} in ${TMP_ROOT}"
            found='true'
        fi
    done
    if ${found}; then
        info "found ${p}"
    else
        error "not found ${p}"
        retcode=1
    fi
done
exit ${retcode}
