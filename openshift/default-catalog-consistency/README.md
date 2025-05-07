Default Catalog Consistency
========================

Those tests are used to check the consistency of the default catalogs in OpenShift.

### Running Catalog Tests Locally

When running the catalog validation tests locally, we bypass container 
signature verification by using an insecure policy file configured 
specifically for development.

To run the tests:

```bash
make test-catalog-local
```