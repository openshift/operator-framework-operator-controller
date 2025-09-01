// Code generated for package testdata by go-bindata DO NOT EDIT. (@generated)
// sources:
// test/extended/testdata/olm/basic-bd-plain-image.yaml
// test/extended/testdata/olm/basic-bd-registry-image.yaml
// test/extended/testdata/olm/binding-prefligth.yaml
// test/extended/testdata/olm/binding-prefligth_multirole.yaml
// test/extended/testdata/olm/cip.yaml
// test/extended/testdata/olm/clustercatalog-secret-withlabel.yaml
// test/extended/testdata/olm/clustercatalog-secret.yaml
// test/extended/testdata/olm/clustercatalog-withlabel.yaml
// test/extended/testdata/olm/clustercatalog.yaml
// test/extended/testdata/olm/clusterextension-withoutChannel-OwnSingle.yaml
// test/extended/testdata/olm/clusterextension-withselectorExpressions-WithoutChannelVersion.yaml
// test/extended/testdata/olm/clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml
// test/extended/testdata/olm/clusterextension-withselectorlabel-OwnSingle.yaml
// test/extended/testdata/olm/clusterextension-withselectorlabel-WithoutChannel.yaml
// test/extended/testdata/olm/clusterextension-withselectorlabel-WithoutChannelVersion.yaml
// test/extended/testdata/olm/clusterextension-withselectorlabel.yaml
// test/extended/testdata/olm/clusterextension.yaml
// test/extended/testdata/olm/clusterextensionWithoutChannel.yaml
// test/extended/testdata/olm/clusterextensionWithoutChannelVersion.yaml
// test/extended/testdata/olm/clusterextensionWithoutVersion.yaml
// test/extended/testdata/olm/crd-nginxolm74923.yaml
// test/extended/testdata/olm/icsp-single-mirror.yaml
// test/extended/testdata/olm/itdms-full-mirror.yaml
// test/extended/testdata/olm/prefligth-clusterrole.yaml
// test/extended/testdata/olm/sa-admin.yaml
// test/extended/testdata/olm/sa-nginx-insufficient-bundle.yaml
// test/extended/testdata/olm/sa-nginx-insufficient-operand-clusterrole.yaml
// test/extended/testdata/olm/sa-nginx-insufficient-operand-rbac.yaml
// test/extended/testdata/olm/sa-nginx-limited.yaml
// test/extended/testdata/olm/sa.yaml
package testdata

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _testExtendedTestdataOlmBasicBdPlainImageYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: basic-bd-image-template
objects:
- apiVersion: core.rukpak.io/v1alpha2
  kind: BundleDeployment
  metadata:
    name: "${NAME}"
  spec:
    installNamespace: "${NAMESPACE}"
    provisionerClassName: "core-rukpak-io-plain"
    source:
      image:
        ref: "${ADDRESS}"
      type: image
parameters:
- name: NAME
- name: ADDRESS
- name: NAMESPACE
`)

func testExtendedTestdataOlmBasicBdPlainImageYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmBasicBdPlainImageYaml, nil
}

func testExtendedTestdataOlmBasicBdPlainImageYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmBasicBdPlainImageYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/basic-bd-plain-image.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmBasicBdRegistryImageYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: basic-bd-image-template
objects:
- apiVersion: core.rukpak.io/v1alpha2
  kind: BundleDeployment
  metadata:
    name: "${NAME}"
  spec:
    installNamespace: "${NAMESPACE}"
    provisionerClassName: "core-rukpak-io-registry"
    source:
      image:
        ref: "${ADDRESS}"
      type: image
parameters:
- name: NAME
- name: ADDRESS
- name: NAMESPACE
`)

func testExtendedTestdataOlmBasicBdRegistryImageYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmBasicBdRegistryImageYaml, nil
}

func testExtendedTestdataOlmBasicBdRegistryImageYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmBasicBdRegistryImageYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/basic-bd-registry-image.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmBindingPrefligthYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-binding-preflight-template
objects:
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${CLUSTERROLESANAME}-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${CLUSTERROLESANAME}"
    subjects:
      - kind: ServiceAccount
        name: "${SANAME}"
        namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      name: "${ROLENAME}-binding"
      namespace: "${NAMESPACE}"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: "${ROLENAME}"
      namespace: "${NAMESPACE}"
    subjects:
      - kind: ServiceAccount
        name: "${SANAME}"
        namespace: "${NAMESPACE}"
parameters:
  - name: SANAME
  - name: ROLENAME
  - name: CLUSTERROLESANAME
  - name: NAMESPACE
`)

func testExtendedTestdataOlmBindingPrefligthYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmBindingPrefligthYaml, nil
}

func testExtendedTestdataOlmBindingPrefligthYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmBindingPrefligthYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/binding-prefligth.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmBindingPrefligth_multiroleYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-binding-preflight-template
objects:
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${CLUSTERROLESANAME}-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${CLUSTERROLESANAME}"
    subjects:
      - kind: ServiceAccount
        name: "${SANAME}"
        namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      name: "${ROLENAME}-binding"
      namespace: "${NAMESPACE}"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: "${ROLENAME}"
      namespace: "${NAMESPACE}"
    subjects:
      - kind: ServiceAccount
        name: "${SANAME}"
        namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      name: "${WATCHROLENAME}-binding"
      namespace: "${WATCHNAMESPACE}"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: "${WATCHROLENAME}"
      namespace: "${WATCHNAMESPACE}"
    subjects:
      - kind: ServiceAccount
        name: "${SANAME}"
        namespace: "${NAMESPACE}"
parameters:
  - name: SANAME
  - name: ROLENAME
  - name: CLUSTERROLESANAME
  - name: NAMESPACE
  - name: WATCHROLENAME
  - name: WATCHNAMESPACE
`)

func testExtendedTestdataOlmBindingPrefligth_multiroleYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmBindingPrefligth_multiroleYaml, nil
}

func testExtendedTestdataOlmBindingPrefligth_multiroleYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmBindingPrefligth_multiroleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/binding-prefligth_multirole.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmCipYaml = []byte(`kind: Template
apiVersion: template.openshift.io/v1
metadata:
  name: cip-template
objects:
- apiVersion: config.openshift.io/v1alpha1
  kind: ClusterImagePolicy
  metadata:
    name: "${NAME}"
  spec:
    policy:
      rootOfTrust:
        policyType: PublicKey
        publicKey: # it is public key, so it is not sensitive information
          keyData: LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KTUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJemowREFRY0RRZ0FFcFFMeTN6VC92WG0yQlZpaFNicmtCWWxXWXJjMwovT1RYYlFkMTIzRFNJdGNBSWFRQlB3dGhqSkNEK01sNzJaTFhIdWZGUnlmek9kRjM3Q3k5OERHV3hRPT0KLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg==
      signedIdentity:
        matchPolicy: "${POLICY}"
    scopes:
    - "${REPO1}"
    - "${REPO2}"
    - "${REPO3}"
    - "${REPO4}"
parameters:
- name: NAME
- name: REPO1
- name: REPO2
- name: REPO3
- name: REPO4
- name: POLICY
  value: "MatchRepoDigestOrExact"
`)

func testExtendedTestdataOlmCipYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmCipYaml, nil
}

func testExtendedTestdataOlmCipYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmCipYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/cip.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClustercatalogSecretWithlabelYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: catalog-secret-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterCatalog
  metadata:
    name: "${NAME}"
    labels:
      "${LABELKEY}: ${LABELVALUE}"
  spec:
    source:
      type: "${TYPE}"
      image:
        pullSecret: "${SECRET}"
        ref: "${IMAGE}"
        pollIntervalMinutes: ${{POLLINTERVALMINUTES}}
parameters:
- name: NAME
- name: TYPE
  value: "Image"
- name: IMAGE
- name: SECRET
- name: POLLINTERVALMINUTES
  value: "60"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
`)

func testExtendedTestdataOlmClustercatalogSecretWithlabelYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClustercatalogSecretWithlabelYaml, nil
}

func testExtendedTestdataOlmClustercatalogSecretWithlabelYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClustercatalogSecretWithlabelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clustercatalog-secret-withlabel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClustercatalogSecretYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: catalog-secret-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterCatalog
  metadata:
    name: "${NAME}"
  spec:
    source:
      type: "${TYPE}"
      image:
        pullSecret: "${SECRET}"
        ref: "${IMAGE}"
        pollIntervalMinutes: ${{POLLINTERVALMINUTES}}
parameters:
- name: NAME
- name: TYPE
  value: "Image"
- name: IMAGE
- name: SECRET
- name: POLLINTERVALMINUTES
  value: "60"
`)

func testExtendedTestdataOlmClustercatalogSecretYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClustercatalogSecretYaml, nil
}

func testExtendedTestdataOlmClustercatalogSecretYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClustercatalogSecretYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clustercatalog-secret.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClustercatalogWithlabelYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: catalog-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterCatalog
  metadata:
    name: "${NAME}"
    labels:
      "${LABELKEY}": "${LABELVALUE}"
  spec:
    source:
      type: "${TYPE}"
      image:
        ref: "${IMAGE}"
parameters:
- name: NAME
- name: TYPE
  value: "Image"
- name: IMAGE
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
`)

func testExtendedTestdataOlmClustercatalogWithlabelYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClustercatalogWithlabelYaml, nil
}

func testExtendedTestdataOlmClustercatalogWithlabelYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClustercatalogWithlabelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clustercatalog-withlabel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClustercatalogYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: catalog-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterCatalog
  metadata:
    name: "${NAME}"
  spec:
    source:
      type: "${TYPE}"
      image:
        ref: "${IMAGE}"
parameters:
- name: NAME
- name: TYPE
  value: "Image"
- name: IMAGE
`)

func testExtendedTestdataOlmClustercatalogYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClustercatalogYaml, nil
}

func testExtendedTestdataOlmClustercatalogYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClustercatalogYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clustercatalog.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionWithoutchannelOwnsingleYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
    annotations:
      olm.operatorframework.io/watch-namespace: "${WATCHNS}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        version: "${VERSION}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: WATCHNS
- name: PACKAGE
- name: VERSION
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionWithoutchannelOwnsingleYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionWithoutchannelOwnsingleYaml, nil
}

func testExtendedTestdataOlmClusterextensionWithoutchannelOwnsingleYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionWithoutchannelOwnsingleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextension-withoutChannel-OwnSingle.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        selector:
          matchExpressions:
          - key: "${EXPRESSIONSKEY}"
            operator: "${EXPRESSIONSOPERATOR}"
            values: 
            - "${EXPRESSIONSVALUE1}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: EXPRESSIONSVALUE1
- name: EXPRESSIONSOPERATOR
  # suggest to use case id
- name: EXPRESSIONSKEY
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml, nil
}

func testExtendedTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextension-withselectorExpressions-WithoutChannelVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        selector:
          matchLabels:
            "${LABELKEY}": "${LABELVALUE}"
          matchExpressions:
          - key: "${EXPRESSIONSKEY}"
            operator: "${EXPRESSIONSOPERATOR}"
            values: 
            - "${EXPRESSIONSVALUE1}"
            - "${EXPRESSIONSVALUE2}"
            - "${EXPRESSIONSVALUE3}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: EXPRESSIONSKEY
- name: EXPRESSIONSOPERATOR
- name: EXPRESSIONSVALUE1
- name: EXPRESSIONSVALUE2
- name: EXPRESSIONSVALUE3
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml, nil
}

func testExtendedTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
    annotations:
      olm.operatorframework.io/watch-namespace: "${WATCHNS}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        channels:
          - "${CHANNEL}"
        version: "${VERSION}"
        selector:
          matchLabels:
            "${LABELKEY}": "${LABELVALUE}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: WATCHNS
- name: PACKAGE
- name: CHANNEL
- name: VERSION
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionWithselectorlabelOwnsingleYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml, nil
}

func testExtendedTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionWithselectorlabelOwnsingleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextension-withselectorlabel-OwnSingle.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        version: "${VERSION}"
        selector:
          matchLabels:
            "${LABELKEY}": "${LABELVALUE}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: VERSION
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml, nil
}

func testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextension-withselectorlabel-WithoutChannel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        selector:
          matchLabels:
            "${LABELKEY}": "${LABELVALUE}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml, nil
}

func testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextension-withselectorlabel-WithoutChannelVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionWithselectorlabelYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        channels:
          - "${CHANNEL}"
        version: "${VERSION}"
        selector:
          matchLabels:
            "${LABELKEY}": "${LABELVALUE}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: CHANNEL
- name: VERSION
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionWithselectorlabelYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionWithselectorlabelYaml, nil
}

func testExtendedTestdataOlmClusterextensionWithselectorlabelYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionWithselectorlabelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextension-withselectorlabel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        channels:
          - "${CHANNEL}"
        version: "${VERSION}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: CHANNEL
- name: VERSION
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionYaml, nil
}

func testExtendedTestdataOlmClusterextensionYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextension.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionwithoutchannelYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-without-channel-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        version: "${VERSION}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: VERSION
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: SOURCETYPE
  value: "Catalog"


`)

func testExtendedTestdataOlmClusterextensionwithoutchannelYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionwithoutchannelYaml, nil
}

func testExtendedTestdataOlmClusterextensionwithoutchannelYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionwithoutchannelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextensionWithoutChannel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionwithoutchannelversionYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-without-channel-version-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: SANAME
- name: SOURCETYPE
  value: "Catalog"



`)

func testExtendedTestdataOlmClusterextensionwithoutchannelversionYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionwithoutchannelversionYaml, nil
}

func testExtendedTestdataOlmClusterextensionwithoutchannelversionYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionwithoutchannelversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextensionWithoutChannelVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmClusterextensionwithoutversionYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-without-channel-template
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    serviceAccount:
      name: "${SANAME}"
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        channels:
          - "${CHANNEL}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: CHANNEL
- name: SANAME
- name: POLICY
  value: "CatalogProvided"
- name: SOURCETYPE
  value: "Catalog"

`)

func testExtendedTestdataOlmClusterextensionwithoutversionYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmClusterextensionwithoutversionYaml, nil
}

func testExtendedTestdataOlmClusterextensionwithoutversionYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmClusterextensionwithoutversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/clusterextensionWithoutVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmCrdNginxolm74923Yaml = []byte(`apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  name: nginxolm74923s.cache.example.com
spec:
  group: cache.example.com
  names:
    kind: Nginxolm74923
    listKind: Nginxolm74923List
    plural: nginxolm74923s
    singular: nginxolm74923
  scope: Namespaced
  versions:
  - name: v1alpha1
    schema:
      openAPIV3Schema:
        description: Nginxolm74923 is the Schema for the nginxolm74923s API
        properties:
          apiVersion:
            description: 'APIVersion defines the versioned schema of this representation
              of an object. Servers should convert recognized schemas to the latest
              internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
            type: string
          kind:
            description: 'Kind is a string value representing the REST resource this
              object represents. Servers may infer this from the endpoint the client
              submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
            type: string
          metadata:
            type: object
          spec:
            description: Spec defines the desired state of Nginxolm74923
            type: object
            x-kubernetes-preserve-unknown-fields: true
          status:
            description: Status defines the observed state of Nginxolm74923
            type: object
            x-kubernetes-preserve-unknown-fields: true
        type: object
    served: true
    storage: true
    subresources:
      status: {}
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: null
  storedVersions: null
`)

func testExtendedTestdataOlmCrdNginxolm74923YamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmCrdNginxolm74923Yaml, nil
}

func testExtendedTestdataOlmCrdNginxolm74923Yaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmCrdNginxolm74923YamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/crd-nginxolm74923.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmIcspSingleMirrorYaml = []byte(`kind: Template
apiVersion: template.openshift.io/v1
metadata:
  name: icsp-single-mirror-template
objects:
- apiVersion: operator.openshift.io/v1alpha1
  kind: ImageContentSourcePolicy
  metadata:
    name: "${NAME}"
  spec:
    repositoryDigestMirrors:
    - mirrors:
      - "${MIRROR}"
      source: "${SOURCE}"
parameters:
- name: NAME
- name: MIRROR
- name: SOURCE
`)

func testExtendedTestdataOlmIcspSingleMirrorYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmIcspSingleMirrorYaml, nil
}

func testExtendedTestdataOlmIcspSingleMirrorYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmIcspSingleMirrorYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/icsp-single-mirror.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmItdmsFullMirrorYaml = []byte(`kind: Template
apiVersion: template.openshift.io/v1
metadata:
  name: itdms-full-mirror-template
objects:
- apiVersion: config.openshift.io/v1
  kind: ImageTagMirrorSet
  metadata:
    name: "${NAME}"
  spec:
    imageTagMirrors:
    - mirrors:
      - "${MIRRORNAMESPACE}"
      source: "${SOURCENAMESPACE}"
    - mirrors:
      - "${MIRRORSITE}"
      source: "${SOURCESITE}"
- apiVersion: config.openshift.io/v1
  kind: ImageDigestMirrorSet
  metadata:
    name: "${NAME}"
  spec:
    imageDigestMirrors:
    - mirrors:
      - "${MIRRORNAMESPACE}"
      source: "${SOURCENAMESPACE}"
    - mirrors:
      - "${MIRRORSITE}"
      source: "${SOURCESITE}"
parameters:
- name: NAME
- name: MIRRORSITE
- name: SOURCESITE
- name: MIRRORNAMESPACE
- name: SOURCENAMESPACE


`)

func testExtendedTestdataOlmItdmsFullMirrorYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmItdmsFullMirrorYaml, nil
}

func testExtendedTestdataOlmItdmsFullMirrorYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmItdmsFullMirrorYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/itdms-full-mirror.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmPrefligthClusterroleYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-preflight-clusterrole-template
objects:
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}"
    rules:
parameters:
  - name: NAME
`)

func testExtendedTestdataOlmPrefligthClusterroleYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmPrefligthClusterroleYaml, nil
}

func testExtendedTestdataOlmPrefligthClusterroleYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmPrefligthClusterroleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/prefligth-clusterrole.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmSaAdminYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-sa-admin-template
objects:
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-admin-clusterrole"
    rules:
      - apiGroups:
        - "*"
        resources:
        - "*"
        verbs:
        - "*"
  - apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-admin-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-admin-clusterrole"
    subjects:
      - kind: ServiceAccount
        name: "${NAME}"
        namespace: "${NAMESPACE}"
parameters:
  - name: NAME
  - name: NAMESPACE

`)

func testExtendedTestdataOlmSaAdminYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmSaAdminYaml, nil
}

func testExtendedTestdataOlmSaAdminYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmSaAdminYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/sa-admin.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmSaNginxInsufficientBundleYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-sa-nginx-insufficient-bundle-template
objects:
  - apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-clusterrole"
    rules:
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterroles]
      verbs: [create]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterroles]
      verbs: [get, list, watch, update, patch, delete]
      # resourceNames:
      # - nginx-ok-v3283-754-15pkpuong3owt1jn01uoyj8lm6p8jlxh03kuouq67dmv
      # - nginx-ok-v3283-754-2r5zqsa9t9nk0tln1f8x36ws3ks9r8cgwi70s2dgnl82
      # - nginx-ok-v3283-75493-metrics-reader
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterrolebindings]
      verbs: [create]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterrolebindings]
      verbs: [get, list, watch, update, patch, delete]
      # resourceNames:
      # - nginx-ok-v3283-754-15pkpuong3owt1jn01uoyj8lm6p8jlxh03kuouq67dmv
      # - nginx-ok-v3283-754-2r5zqsa9t9nk0tln1f8x36ws3ks9r8cgwi70s2dgnl82
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-clusterrole"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: Role
    metadata:
      name: "${NAME}-installer-role"
      namespace: "${NAMESPACE}"
    rules:
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [get, list, watch, create, update, patch, delete]
    #   resourceNames: [nginx-ok-v3283-75493-controller-manager]
    # - apiGroups: [""]
    #   resources: [serviceaccounts]
    #   verbs: [create]
    - apiGroups: [""]
      resources: [services]
      verbs: [get, list, watch, create, update, patch, delete]
      # resourceNames: [nginx-ok-v3283-75493-controller-manager-metrics-service]
    - apiGroups: [""]
      resources: [services]
      verbs: [create]
    - apiGroups: [apps]
      resources: [deployments]
      verbs: [get, list, watch, create, update, patch, delete]
      # resourceNames: [nginx-ok-v3283-75493-controller-manager]
    - apiGroups: [apps]
      resources: [deployments]
      verbs: [create]
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      name: "${NAME}-installer-role-binding"
      namespace: "${NAMESPACE}"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: "${NAME}-installer-role"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-rbac-clusterrole"
    rules:
    - apiGroups:
      - ""
      resources:
      - configmaps
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    - apiGroups:
      - coordination.k8s.io
      resources:
      - leases
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    - apiGroups:
      - ""
      resources:
      - events
      verbs:
      - create
      - patch
    - apiGroups:
      - ""
      resources:
      - secrets
      - pods
      - pods/exec
      - pods/log
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - apps
      resources:
      - deployments
      - daemonsets
      - replicasets
      - statefulsets
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - cache.example.com
      resources:
      - "${KINDS}"
      - "${KINDS}/status"
      - "${KINDS}/finalizers"
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - authentication.k8s.io
      resources:
      - tokenreviews
      verbs:
      - create
    - apiGroups:
      - authorization.k8s.io
      resources:
      - subjectaccessreviews
      verbs:
      - create
    - nonResourceURLs:
      - /metrics
      verbs:
      - get
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-rbac-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-rbac-clusterrole"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
parameters:
  - name: NAME
  - name: NAMESPACE
  - name: KINDS
`)

func testExtendedTestdataOlmSaNginxInsufficientBundleYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmSaNginxInsufficientBundleYaml, nil
}

func testExtendedTestdataOlmSaNginxInsufficientBundleYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmSaNginxInsufficientBundleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/sa-nginx-insufficient-bundle.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmSaNginxInsufficientOperandClusterroleYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-sa-nginx-insufficient-operand-clusterrole-template
objects:
  - apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-clusterrole"
    rules:
    - apiGroups: [apiextensions.k8s.io]
      resources: [customresourcedefinitions]
      verbs: [create, list, watch]
    - apiGroups: [apiextensions.k8s.io]
      resources: [customresourcedefinitions]
      verbs: [get, update, patch, delete]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterroles]
      verbs: [create]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterroles]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterrolebindings]
      verbs: [create]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterrolebindings]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [get, list, watch, create, update, patch, delete]
    - apiGroups: [""]
      resources: [services]
      verbs: [get, list, watch, create, update, patch, delete]
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-clusterrole"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: Role
    metadata:
      name: "${NAME}-installer-role"
      namespace: "${NAMESPACE}"
    rules:
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [create]
    - apiGroups: [""]
      resources: [services]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [""]
      resources: [services]
      verbs: [create]
    - apiGroups: [apps]
      resources: [deployments]
      verbs: [get, list, watch, create, update, patch, delete]
    - apiGroups: [apps]
      resources: [deployments]
      verbs: [create]
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      name: "${NAME}-installer-role-binding"
      namespace: "${NAMESPACE}"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: "${NAME}-installer-role"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-rbac-clusterrole"
    rules:
    - apiGroups:
      - ""
      resources:
      - configmaps
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    - apiGroups:
      - coordination.k8s.io
      resources:
      - leases
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    - apiGroups:
      - ""
      resources:
      - events
      verbs:
      - create
      - patch
    - apiGroups:
      - apps
      resources:
      - deployments
      - daemonsets
      - replicasets
      - statefulsets
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - cache.example.com
      resources:
      - "${KINDS}"
      - "${KINDS}/status"
      - "${KINDS}/finalizers"
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - authentication.k8s.io
      resources:
      - tokenreviews
      verbs:
      - create
    - apiGroups:
      - authorization.k8s.io
      resources:
      - subjectaccessreviews
      verbs:
      - create
    - nonResourceURLs:
      - /metrics
      verbs:
      - get
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-rbac-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-rbac-clusterrole"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
parameters:
  - name: NAME
  - name: NAMESPACE
  - name: KINDS
`)

func testExtendedTestdataOlmSaNginxInsufficientOperandClusterroleYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmSaNginxInsufficientOperandClusterroleYaml, nil
}

func testExtendedTestdataOlmSaNginxInsufficientOperandClusterroleYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmSaNginxInsufficientOperandClusterroleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/sa-nginx-insufficient-operand-clusterrole.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmSaNginxInsufficientOperandRbacYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-sa-nginx-insufficient-operand-rbac-template
objects:
  - apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-clusterrole"
    rules:
    - apiGroups: [olm.operatorframework.io]
      resources: [clusterextensions/finalizers]
      verbs: [update]
    - apiGroups: [apiextensions.k8s.io]
      resources: [customresourcedefinitions]
      verbs: [create, list, watch]
    - apiGroups: [apiextensions.k8s.io]
      resources: [customresourcedefinitions]
      verbs: [get, update, patch, delete]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterroles]
      verbs: [create]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterroles]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterrolebindings]
      verbs: [create]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterrolebindings]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [get, list, watch, create, update, patch, delete]
    - apiGroups: [""]
      resources: [services]
      verbs: [get, list, watch, create, update, patch, delete]
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-clusterrole"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: Role
    metadata:
      name: "${NAME}-installer-role"
      namespace: "${NAMESPACE}"
    rules:
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [create]
    - apiGroups: [""]
      resources: [services]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [""]
      resources: [services]
      verbs: [create]
    - apiGroups: [apps]
      resources: [deployments]
      verbs: [get, list, watch, create, update, patch, delete]
    - apiGroups: [apps]
      resources: [deployments]
      verbs: [create]
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      name: "${NAME}-installer-role-binding"
      namespace: "${NAMESPACE}"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: "${NAME}-installer-role"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-rbac-clusterrole"
    rules:
    - apiGroups:
      - ""
      resources:
      - configmaps
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    - apiGroups:
      - coordination.k8s.io
      resources:
      - leases
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    - apiGroups:
      - ""
      resources:
      - events
      verbs:
      - create
      - patch
    - apiGroups:
      - apps
      resources:
      - deployments
      - daemonsets
      - replicasets
      - statefulsets
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - cache.example.com
      resources:
      - "${KINDS}"
      - "${KINDS}/status"
      - "${KINDS}/finalizers"
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - authentication.k8s.io
      resources:
      - tokenreviews
      verbs:
      - create
    - apiGroups:
      - authorization.k8s.io
      resources:
      - subjectaccessreviews
      verbs:
      - create
    - nonResourceURLs:
      - /metrics
      verbs:
      - get
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-rbac-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-rbac-clusterrole"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
parameters:
  - name: NAME
  - name: NAMESPACE
  - name: KINDS
`)

func testExtendedTestdataOlmSaNginxInsufficientOperandRbacYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmSaNginxInsufficientOperandRbacYaml, nil
}

func testExtendedTestdataOlmSaNginxInsufficientOperandRbacYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmSaNginxInsufficientOperandRbacYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/sa-nginx-insufficient-operand-rbac.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmSaNginxLimitedYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-sa-nginx-limited-template
objects:
  - apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-clusterrole"
    rules:
    - apiGroups: [olm.operatorframework.io]
      resources: [clusterextensions/finalizers]
      verbs: [update]
    - apiGroups: [apiextensions.k8s.io]
      resources: [customresourcedefinitions]
      verbs: [create, list, watch]
    - apiGroups: [apiextensions.k8s.io]
      resources: [customresourcedefinitions]
      verbs: [get, update, patch, delete]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterroles]
      verbs: [create]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterroles]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterrolebindings]
      verbs: [create]
    - apiGroups: [rbac.authorization.k8s.io]
      resources: [clusterrolebindings]
      verbs: [get, list, watch, update, patch, delete]
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [get, list, watch, create, update, patch, delete]
    - apiGroups: [""]
      resources: [services]
      verbs: [get, list, watch, create, update, patch, delete]
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-clusterrole"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: Role
    metadata:
      name: "${NAME}-installer-role"
      namespace: "${NAMESPACE}"
    rules:
    - apiGroups: [""]
      resources: [serviceaccounts]
      verbs: [get, list, watch, create, update, patch, delete]
    - apiGroups: [""]
      resources: [services]
      verbs: [get, list, watch, create, update, patch, delete]
    - apiGroups: [apps]
      resources: [deployments]
      verbs: [get, list, watch, create, update, patch, delete]
    - apiGroups: [apps]
      resources: [deployments]
      verbs: [create]
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: RoleBinding
    metadata:
      name: "${NAME}-installer-role-binding"
      namespace: "${NAMESPACE}"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: Role
      name: "${NAME}-installer-role"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRole
    metadata:
      name: "${NAME}-installer-rbac-clusterrole"
    rules:
    - apiGroups:
      - ""
      resources:
      - namespaces
      verbs:
      - get
      - list
      - watch
    - apiGroups:
      - ""
      resources:
      - configmaps
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    - apiGroups:
      - coordination.k8s.io
      resources:
      - leases
      verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
    - apiGroups:
      - ""
      resources:
      - events
      verbs:
      - create
      - patch
    - apiGroups:
      - ""
      resources:
      - secrets
      - pods
      - pods/exec
      - pods/log
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - apps
      resources:
      - deployments
      - daemonsets
      - replicasets
      - statefulsets
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - cache.example.com
      resources:
      - "${KINDS}"
      - "${KINDS}/status"
      - "${KINDS}/finalizers"
      verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
    - apiGroups:
      - authentication.k8s.io
      resources:
      - tokenreviews
      verbs:
      - create
    - apiGroups:
      - authorization.k8s.io
      resources:
      - subjectaccessreviews
      verbs:
      - create
    - nonResourceURLs:
      - /metrics
      verbs:
      - get
  - apiVersion: rbac.authorization.k8s.io/v1
    kind: ClusterRoleBinding
    metadata:
      name: "${NAME}-installer-rbac-clusterrole-binding"
    roleRef:
      apiGroup: rbac.authorization.k8s.io
      kind: ClusterRole
      name: "${NAME}-installer-rbac-clusterrole"
    subjects:
    - kind: ServiceAccount
      name: "${NAME}"
      namespace: "${NAMESPACE}"
parameters:
  - name: NAME
  - name: NAMESPACE
  - name: KINDS
`)

func testExtendedTestdataOlmSaNginxLimitedYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmSaNginxLimitedYaml, nil
}

func testExtendedTestdataOlmSaNginxLimitedYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmSaNginxLimitedYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/sa-nginx-limited.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testExtendedTestdataOlmSaYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: olmv1-sa-template
objects:
  - apiVersion: v1
    kind: ServiceAccount
    metadata:
      name: "${NAME}"
      namespace: "${NAMESPACE}"
parameters:
  - name: NAME
  - name: NAMESPACE
`)

func testExtendedTestdataOlmSaYamlBytes() ([]byte, error) {
	return _testExtendedTestdataOlmSaYaml, nil
}

func testExtendedTestdataOlmSaYaml() (*asset, error) {
	bytes, err := testExtendedTestdataOlmSaYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/extended/testdata/olm/sa.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"test/extended/testdata/olm/basic-bd-plain-image.yaml":                                                testExtendedTestdataOlmBasicBdPlainImageYaml,
	"test/extended/testdata/olm/basic-bd-registry-image.yaml":                                             testExtendedTestdataOlmBasicBdRegistryImageYaml,
	"test/extended/testdata/olm/binding-prefligth.yaml":                                                   testExtendedTestdataOlmBindingPrefligthYaml,
	"test/extended/testdata/olm/binding-prefligth_multirole.yaml":                                         testExtendedTestdataOlmBindingPrefligth_multiroleYaml,
	"test/extended/testdata/olm/cip.yaml":                                                                 testExtendedTestdataOlmCipYaml,
	"test/extended/testdata/olm/clustercatalog-secret-withlabel.yaml":                                     testExtendedTestdataOlmClustercatalogSecretWithlabelYaml,
	"test/extended/testdata/olm/clustercatalog-secret.yaml":                                               testExtendedTestdataOlmClustercatalogSecretYaml,
	"test/extended/testdata/olm/clustercatalog-withlabel.yaml":                                            testExtendedTestdataOlmClustercatalogWithlabelYaml,
	"test/extended/testdata/olm/clustercatalog.yaml":                                                      testExtendedTestdataOlmClustercatalogYaml,
	"test/extended/testdata/olm/clusterextension-withoutChannel-OwnSingle.yaml":                           testExtendedTestdataOlmClusterextensionWithoutchannelOwnsingleYaml,
	"test/extended/testdata/olm/clusterextension-withselectorExpressions-WithoutChannelVersion.yaml":      testExtendedTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml,
	"test/extended/testdata/olm/clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml": testExtendedTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml,
	"test/extended/testdata/olm/clusterextension-withselectorlabel-OwnSingle.yaml":                        testExtendedTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml,
	"test/extended/testdata/olm/clusterextension-withselectorlabel-WithoutChannel.yaml":                   testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml,
	"test/extended/testdata/olm/clusterextension-withselectorlabel-WithoutChannelVersion.yaml":            testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml,
	"test/extended/testdata/olm/clusterextension-withselectorlabel.yaml":                                  testExtendedTestdataOlmClusterextensionWithselectorlabelYaml,
	"test/extended/testdata/olm/clusterextension.yaml":                                                    testExtendedTestdataOlmClusterextensionYaml,
	"test/extended/testdata/olm/clusterextensionWithoutChannel.yaml":                                      testExtendedTestdataOlmClusterextensionwithoutchannelYaml,
	"test/extended/testdata/olm/clusterextensionWithoutChannelVersion.yaml":                               testExtendedTestdataOlmClusterextensionwithoutchannelversionYaml,
	"test/extended/testdata/olm/clusterextensionWithoutVersion.yaml":                                      testExtendedTestdataOlmClusterextensionwithoutversionYaml,
	"test/extended/testdata/olm/crd-nginxolm74923.yaml":                                                   testExtendedTestdataOlmCrdNginxolm74923Yaml,
	"test/extended/testdata/olm/icsp-single-mirror.yaml":                                                  testExtendedTestdataOlmIcspSingleMirrorYaml,
	"test/extended/testdata/olm/itdms-full-mirror.yaml":                                                   testExtendedTestdataOlmItdmsFullMirrorYaml,
	"test/extended/testdata/olm/prefligth-clusterrole.yaml":                                               testExtendedTestdataOlmPrefligthClusterroleYaml,
	"test/extended/testdata/olm/sa-admin.yaml":                                                            testExtendedTestdataOlmSaAdminYaml,
	"test/extended/testdata/olm/sa-nginx-insufficient-bundle.yaml":                                        testExtendedTestdataOlmSaNginxInsufficientBundleYaml,
	"test/extended/testdata/olm/sa-nginx-insufficient-operand-clusterrole.yaml":                           testExtendedTestdataOlmSaNginxInsufficientOperandClusterroleYaml,
	"test/extended/testdata/olm/sa-nginx-insufficient-operand-rbac.yaml":                                  testExtendedTestdataOlmSaNginxInsufficientOperandRbacYaml,
	"test/extended/testdata/olm/sa-nginx-limited.yaml":                                                    testExtendedTestdataOlmSaNginxLimitedYaml,
	"test/extended/testdata/olm/sa.yaml":                                                                  testExtendedTestdataOlmSaYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//
//	data/
//	  foo.txt
//	  img/
//	    a.png
//	    b.png
//
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"test": {nil, map[string]*bintree{
		"extended": {nil, map[string]*bintree{
			"testdata": {nil, map[string]*bintree{
				"olm": {nil, map[string]*bintree{
					"basic-bd-plain-image.yaml":                      {testExtendedTestdataOlmBasicBdPlainImageYaml, map[string]*bintree{}},
					"basic-bd-registry-image.yaml":                   {testExtendedTestdataOlmBasicBdRegistryImageYaml, map[string]*bintree{}},
					"binding-prefligth.yaml":                         {testExtendedTestdataOlmBindingPrefligthYaml, map[string]*bintree{}},
					"binding-prefligth_multirole.yaml":               {testExtendedTestdataOlmBindingPrefligth_multiroleYaml, map[string]*bintree{}},
					"cip.yaml":                                       {testExtendedTestdataOlmCipYaml, map[string]*bintree{}},
					"clustercatalog-secret-withlabel.yaml":           {testExtendedTestdataOlmClustercatalogSecretWithlabelYaml, map[string]*bintree{}},
					"clustercatalog-secret.yaml":                     {testExtendedTestdataOlmClustercatalogSecretYaml, map[string]*bintree{}},
					"clustercatalog-withlabel.yaml":                  {testExtendedTestdataOlmClustercatalogWithlabelYaml, map[string]*bintree{}},
					"clustercatalog.yaml":                            {testExtendedTestdataOlmClustercatalogYaml, map[string]*bintree{}},
					"clusterextension-withoutChannel-OwnSingle.yaml": {testExtendedTestdataOlmClusterextensionWithoutchannelOwnsingleYaml, map[string]*bintree{}},
					"clusterextension-withselectorExpressions-WithoutChannelVersion.yaml":      {testExtendedTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml, map[string]*bintree{}},
					"clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml": {testExtendedTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-OwnSingle.yaml":                        {testExtendedTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-WithoutChannel.yaml":                   {testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-WithoutChannelVersion.yaml":            {testExtendedTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel.yaml":                                  {testExtendedTestdataOlmClusterextensionWithselectorlabelYaml, map[string]*bintree{}},
					"clusterextension.yaml":                          {testExtendedTestdataOlmClusterextensionYaml, map[string]*bintree{}},
					"clusterextensionWithoutChannel.yaml":            {testExtendedTestdataOlmClusterextensionwithoutchannelYaml, map[string]*bintree{}},
					"clusterextensionWithoutChannelVersion.yaml":     {testExtendedTestdataOlmClusterextensionwithoutchannelversionYaml, map[string]*bintree{}},
					"clusterextensionWithoutVersion.yaml":            {testExtendedTestdataOlmClusterextensionwithoutversionYaml, map[string]*bintree{}},
					"crd-nginxolm74923.yaml":                         {testExtendedTestdataOlmCrdNginxolm74923Yaml, map[string]*bintree{}},
					"icsp-single-mirror.yaml":                        {testExtendedTestdataOlmIcspSingleMirrorYaml, map[string]*bintree{}},
					"itdms-full-mirror.yaml":                         {testExtendedTestdataOlmItdmsFullMirrorYaml, map[string]*bintree{}},
					"prefligth-clusterrole.yaml":                     {testExtendedTestdataOlmPrefligthClusterroleYaml, map[string]*bintree{}},
					"sa-admin.yaml":                                  {testExtendedTestdataOlmSaAdminYaml, map[string]*bintree{}},
					"sa-nginx-insufficient-bundle.yaml":              {testExtendedTestdataOlmSaNginxInsufficientBundleYaml, map[string]*bintree{}},
					"sa-nginx-insufficient-operand-clusterrole.yaml": {testExtendedTestdataOlmSaNginxInsufficientOperandClusterroleYaml, map[string]*bintree{}},
					"sa-nginx-insufficient-operand-rbac.yaml":        {testExtendedTestdataOlmSaNginxInsufficientOperandRbacYaml, map[string]*bintree{}},
					"sa-nginx-limited.yaml":                          {testExtendedTestdataOlmSaNginxLimitedYaml, map[string]*bintree{}},
					"sa.yaml":                                        {testExtendedTestdataOlmSaYaml, map[string]*bintree{}},
				}},
			}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
