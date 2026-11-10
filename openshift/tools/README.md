# OpenShift build tools

This module pins the command-line tools used under `openshift/`. Run tools through
`run.sh` so they execute in the caller's directory while `go tool` resolves them
from this module.

To update a tool, run `go get -tool <package>@<version>`, `go mod tidy`, and
`go mod vendor` in this directory. The shared module resolves one version of
each dependency across its tools. Helm v3.18.4 requires Kustomize v5.6.0, so
the Kustomize tool uses v5.6.0.
