# Embedded Third-Party Code: crdify

This directory contains an embedded copy of the `sigs.k8s.io/crdify` package, which is used for performing CRD upgrade safety checks in the operator-controller.

## Why This Code Is Embedded

We've embedded this third-party code directly into our repository for the following reasons:

1. **Ease of Maintenance**: By embedding the code, we maintain full control over the version and can ensure compatibility with our current codebase without being affected by upstream changes that might introduce breaking changes or require dependency updates.

2. **Cherry-Pick Bug Fixes**: Having the code in our repository allows us to easily cherry-pick specific bug fixes from upstream without needing to upgrade to a newer version that might have other incompatibilities or require Go version upgrades.

3. **Go Version Compatibility**: The newer versions of `crdify` (v0.5.0+) require Go 1.24+, which is not supported in our current environment. By embedding a compatible version, we can continue using the functionality without needing to upgrade our Go toolchain.

## Version Information

- **Embedded Version**: `v0.4.1-0.20250613143457-398e4483fb58`
- **Source Repository**: `sigs.k8s.io/crdify`
- **License**: Apache License 2.0 (see [LICENSE](./LICENSE))

This version was ported from the `release-4.20` branch and is compatible with our current Go version requirements.

## Usage

This embedded code is used by the CRD upgrade safety preflight checks located in:
- `internal/operator-controller/rukpak/preflights/crdupgradesafety/`

The code has been adapted to use local imports instead of the upstream package:
- `sigs.k8s.io/crdify/pkg/config` → `github.com/operator-framework/operator-controller/internal/thirdparty/crdify/pkg/config`
- `sigs.k8s.io/crdify/pkg/runner` → `github.com/operator-framework/operator-controller/internal/thirdparty/crdify/pkg/runner`
- `sigs.k8s.io/crdify/pkg/validations` → `github.com/operator-framework/operator-controller/internal/thirdparty/crdify/pkg/validations`
- `sigs.k8s.io/crdify/pkg/validations/property` → `github.com/operator-framework/operator-controller/internal/thirdparty/crdify/pkg/validations/property`

## Updating This Code

If you need to update this embedded code:

1. Identify the specific version or commit from upstream that you want to use
2. Ensure it's compatible with our Go version requirements
3. Copy the updated code into this directory
4. Update the import paths in the consuming code
5. Update this README with the new version information
6. Test thoroughly to ensure compatibility

## Verification

To verify that the port from `release-4.20` is complete and only imports have changed:

```bash
git diff upstream/release-4.20 -- internal/operator-controller/rukpak/preflights/crdupgradesafety
```

To verify that all testdata has been ported:

```bash
git diff upstream/release-4.20 -- internal/operator-controller/rukpak/preflights/crdupgradesafety/testdata/manifests
```
