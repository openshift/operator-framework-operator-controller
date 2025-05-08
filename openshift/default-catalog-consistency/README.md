Default Catalog Consistency
========================

Those tests are used to check the consistency of the default catalogs in OpenShift.

### Running Tests Without Signature Verification

When running the catalog validation tests locally, we can bypass container 
signature verification by using an insecure policy file configured 
specifically for development.

```bash
make test-catalog-insecure
```