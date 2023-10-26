Default Catalog Consistency
========================

Those tests are used to check the consistency of the default catalogs in OpenShift.

### Running Tests

```bash
make test-catalog
```

### Before Pushing Changes

Before pushing changes to the repository, you should verify that the tests pass. 
This can be done by running:

```bash
make verify
```