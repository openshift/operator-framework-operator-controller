# How to Run the Origin Tests Against a Cluster

## Create a Cluster

The origin tests will run against any OpenShift cluster, you just need to set the
`KUBECONFIG` variable for the `oc` (or `kubectl`) command.

Additional tests will be run on a `techpreview` deployment.

### Example Clusterbot Command
```
launch 4.20 aws,techpreview
```

## Get the openshift/origin Repo

```
git clone https://github.com/openshift/origin.git
```

## Build the Tests

From the root of the repository:
```
make WHAT=cmd/openshift-tests
```

## List the OLMv1 Tests

All the OLMv1 tests use a `sig-olmv1` prefix, and this can be used to minimize the set of tests. From within the root of the `openshift/origin` repo:

```
./openshift-tests run all --dry-run | grep sig-olmv1
```

## Run ALL the Tests

To run all the tests, use the `run all` options. This requires the `KUBECONFIG` variable to reference the OpenShift cluster.

```
./openshift-tests run all
```

## Run a Subset (e.g. OLMv1) of the Tests

To run a subset of the tests, use the `-f` option with the desrired tests. Passing `-` to the `-f` argument will let you specify the tests via stdin.

```
./openshift-tests run all --dry-run | grep sig-olmv1 | ./openshift-tests run -f -
```
