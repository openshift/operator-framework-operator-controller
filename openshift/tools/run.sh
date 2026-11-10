#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

TOOLS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
tool="$1"
shift

# Resolve the tool using the shared module, then run it in the caller's directory.
tool_path="$(go -C "${TOOLS_DIR}" tool -n "${tool}")"
exec "${tool_path}" "$@"
