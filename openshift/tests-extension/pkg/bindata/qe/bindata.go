// Code generated for package testdata by go-bindata DO NOT EDIT. (@generated)
// sources:
// test/qe/testdata/olm/basic-bd-plain-image.yaml
// test/qe/testdata/olm/basic-bd-registry-image.yaml
// test/qe/testdata/olm/cip.yaml
// test/qe/testdata/olm/clustercatalog-secret-withlabel.yaml
// test/qe/testdata/olm/clustercatalog-secret.yaml
// test/qe/testdata/olm/clustercatalog-with-pollinterval.yaml
// test/qe/testdata/olm/clustercatalog-withlabel.yaml
// test/qe/testdata/olm/clustercatalog.yaml
// test/qe/testdata/olm/clusterextension-watchns-config.yaml
// test/qe/testdata/olm/clusterextension-withselectorExpressions-WithoutChannelVersion.yaml
// test/qe/testdata/olm/clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml
// test/qe/testdata/olm/clusterextension-withselectorlabel-OwnSingle.yaml
// test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutChannel.yaml
// test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutChannelVersion.yaml
// test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutVersion.yaml
// test/qe/testdata/olm/clusterextension-withselectorlabel-inlineconfig.yaml
// test/qe/testdata/olm/clusterextension-withselectorlabel-withoutChannel-OwnSingle.yaml
// test/qe/testdata/olm/clusterextension-withselectorlabel.yaml
// test/qe/testdata/olm/clusterextension.yaml
// test/qe/testdata/olm/clusterextensionWithoutChannel.yaml
// test/qe/testdata/olm/clusterextensionWithoutChannelVersion.yaml
// test/qe/testdata/olm/clusterextensionWithoutVersion.yaml
// test/qe/testdata/olm/cr-webhookTest.yaml
// test/qe/testdata/olm/crd-nginxolm74923.yaml
// test/qe/testdata/olm/icsp-single-mirror.yaml
// test/qe/testdata/olm/itdms-full-mirror.yaml
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

var _testQeTestdataOlmBasicBdPlainImageYaml = []byte(`apiVersion: template.openshift.io/v1
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

func testQeTestdataOlmBasicBdPlainImageYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmBasicBdPlainImageYaml, nil
}

func testQeTestdataOlmBasicBdPlainImageYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmBasicBdPlainImageYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/basic-bd-plain-image.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmBasicBdRegistryImageYaml = []byte(`apiVersion: template.openshift.io/v1
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

func testQeTestdataOlmBasicBdRegistryImageYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmBasicBdRegistryImageYaml, nil
}

func testQeTestdataOlmBasicBdRegistryImageYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmBasicBdRegistryImageYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/basic-bd-registry-image.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmCipYaml = []byte(`kind: Template
apiVersion: template.openshift.io/v1
metadata:
  name: cip-template
objects:
- apiVersion: config.openshift.io/v1
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

func testQeTestdataOlmCipYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmCipYaml, nil
}

func testQeTestdataOlmCipYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmCipYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/cip.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClustercatalogSecretWithlabelYaml = []byte(`apiVersion: template.openshift.io/v1
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

func testQeTestdataOlmClustercatalogSecretWithlabelYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClustercatalogSecretWithlabelYaml, nil
}

func testQeTestdataOlmClustercatalogSecretWithlabelYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClustercatalogSecretWithlabelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clustercatalog-secret-withlabel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClustercatalogSecretYaml = []byte(`apiVersion: template.openshift.io/v1
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

func testQeTestdataOlmClustercatalogSecretYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClustercatalogSecretYaml, nil
}

func testQeTestdataOlmClustercatalogSecretYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClustercatalogSecretYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clustercatalog-secret.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClustercatalogWithPollintervalYaml = []byte(`apiVersion: template.openshift.io/v1
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
        pollInterval: "${POLLINTERVALMINUTES}"
parameters:
- name: NAME
- name: TYPE
  value: "Image"
- name: IMAGE
- name: POLLINTERVALMINUTES
  value: "300s"
`)

func testQeTestdataOlmClustercatalogWithPollintervalYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClustercatalogWithPollintervalYaml, nil
}

func testQeTestdataOlmClustercatalogWithPollintervalYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClustercatalogWithPollintervalYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clustercatalog-with-pollinterval.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClustercatalogWithlabelYaml = []byte(`apiVersion: template.openshift.io/v1
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

func testQeTestdataOlmClustercatalogWithlabelYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClustercatalogWithlabelYaml, nil
}

func testQeTestdataOlmClustercatalogWithlabelYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClustercatalogWithlabelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clustercatalog-withlabel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClustercatalogYaml = []byte(`apiVersion: template.openshift.io/v1
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

func testQeTestdataOlmClustercatalogYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClustercatalogYaml, nil
}

func testQeTestdataOlmClustercatalogYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClustercatalogYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clustercatalog.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWatchnsConfigYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template-watchns-config
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    config:
      configType: Inline
      inline:
        watchNamespace: "${WATCHNS}"
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
- name: WATCHNS
  value: ""
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"
`)

func testQeTestdataOlmClusterextensionWatchnsConfigYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWatchnsConfigYaml, nil
}

func testQeTestdataOlmClusterextensionWatchnsConfigYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWatchnsConfigYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-watchns-config.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml = []byte(`apiVersion: template.openshift.io/v1
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
- name: POLICY
  value: "CatalogProvided"
- name: EXPRESSIONSVALUE1
- name: EXPRESSIONSOPERATOR
  # suggest to use case id
- name: EXPRESSIONSKEY
- name: SOURCETYPE
  value: "Catalog"

`)

func testQeTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorExpressions-WithoutChannelVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml = []byte(`apiVersion: template.openshift.io/v1
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

func testQeTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml = []byte(`apiVersion: template.openshift.io/v1
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
    config:
      configType: Inline
      inline:
        watchNamespace: "${WATCHNS}"
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
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"

`)

func testQeTestdataOlmClusterextensionWithselectorlabelOwnsingleYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorlabelOwnsingleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorlabel-OwnSingle.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml = []byte(`apiVersion: template.openshift.io/v1
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
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"

`)

func testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutChannel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml = []byte(`apiVersion: template.openshift.io/v1
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
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"

`)

func testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutChannelVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorlabelWithoutversionYaml = []byte(`apiVersion: template.openshift.io/v1
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
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
        channels:
          - "${CHANNEL}"
        selector:
          matchLabels:
            "${LABELKEY}": "${LABELVALUE}"
        upgradeConstraintPolicy: "${POLICY}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: CHANNEL
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"
`)

func testQeTestdataOlmClusterextensionWithselectorlabelWithoutversionYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorlabelWithoutversionYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorlabelWithoutversionYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorlabelWithoutversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorlabelInlineconfigYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: operator-template-deploymentconfig
objects:
- apiVersion: olm.operatorframework.io/v1
  kind: ClusterExtension
  metadata:
    name: "${NAME}"
  spec:
    namespace: "${INSTALLNAMESPACE}"
    config:
      configType: Inline
      inline:
        ${{INLINECONFIG}}
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
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"
- name: INLINECONFIG
  description: "JSON object for inline config (must be valid JSON)"
`)

func testQeTestdataOlmClusterextensionWithselectorlabelInlineconfigYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorlabelInlineconfigYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorlabelInlineconfigYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorlabelInlineconfigYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorlabel-inlineconfig.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelOwnsingleYaml = []byte(`apiVersion: template.openshift.io/v1
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
    config:
      configType: Inline
      inline:
        watchNamespace: "${WATCHNS}"
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
- name: WATCHNS
- name: PACKAGE
- name: VERSION
- name: POLICY
  value: "CatalogProvided"
- name: SOURCETYPE
  value: "Catalog"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"




`)

func testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelOwnsingleYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelOwnsingleYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelOwnsingleYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelOwnsingleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorlabel-withoutChannel-OwnSingle.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionWithselectorlabelYaml = []byte(`apiVersion: template.openshift.io/v1
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
- name: POLICY
  value: "CatalogProvided"
- name: LABELVALUE
  # suggest to use case id
- name: LABELKEY
  value: "olmv1-test"
- name: SOURCETYPE
  value: "Catalog"

`)

func testQeTestdataOlmClusterextensionWithselectorlabelYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionWithselectorlabelYaml, nil
}

func testQeTestdataOlmClusterextensionWithselectorlabelYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionWithselectorlabelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension-withselectorlabel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionYaml = []byte(`apiVersion: template.openshift.io/v1
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
- name: POLICY
  value: "CatalogProvided"
- name: SOURCETYPE
  value: "Catalog"

`)

func testQeTestdataOlmClusterextensionYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionYaml, nil
}

func testQeTestdataOlmClusterextensionYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextension.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionwithoutchannelYaml = []byte(`apiVersion: template.openshift.io/v1
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
- name: POLICY
  value: "CatalogProvided"
- name: SOURCETYPE
  value: "Catalog"


`)

func testQeTestdataOlmClusterextensionwithoutchannelYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionwithoutchannelYaml, nil
}

func testQeTestdataOlmClusterextensionwithoutchannelYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionwithoutchannelYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextensionWithoutChannel.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionwithoutchannelversionYaml = []byte(`apiVersion: template.openshift.io/v1
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
    source:
      sourceType: "${SOURCETYPE}"
      catalog:
        packageName: "${PACKAGE}"
parameters:
- name: NAME
- name: INSTALLNAMESPACE
- name: PACKAGE
- name: SOURCETYPE
  value: "Catalog"



`)

func testQeTestdataOlmClusterextensionwithoutchannelversionYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionwithoutchannelversionYaml, nil
}

func testQeTestdataOlmClusterextensionwithoutchannelversionYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionwithoutchannelversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextensionWithoutChannelVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmClusterextensionwithoutversionYaml = []byte(`apiVersion: template.openshift.io/v1
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
- name: POLICY
  value: "CatalogProvided"
- name: SOURCETYPE
  value: "Catalog"

`)

func testQeTestdataOlmClusterextensionwithoutversionYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmClusterextensionwithoutversionYaml, nil
}

func testQeTestdataOlmClusterextensionwithoutversionYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmClusterextensionwithoutversionYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/clusterextensionWithoutVersion.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmCrWebhooktestYaml = []byte(`apiVersion: template.openshift.io/v1
kind: Template
metadata:
  name: webhooktest-template
objects:
  - apiVersion: webhook.operators.coreos.io/v1
    kind: WebhookTest
    metadata:
      name: ${NAME}
      namespace: ${NAMESPACE}
    spec:
      valid: ${{VALID}}
parameters:
  - name: NAME
  - name: NAMESPACE
  - name: VALID
`)

func testQeTestdataOlmCrWebhooktestYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmCrWebhooktestYaml, nil
}

func testQeTestdataOlmCrWebhooktestYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmCrWebhooktestYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/cr-webhookTest.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmCrdNginxolm74923Yaml = []byte(`apiVersion: apiextensions.k8s.io/v1
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

func testQeTestdataOlmCrdNginxolm74923YamlBytes() ([]byte, error) {
	return _testQeTestdataOlmCrdNginxolm74923Yaml, nil
}

func testQeTestdataOlmCrdNginxolm74923Yaml() (*asset, error) {
	bytes, err := testQeTestdataOlmCrdNginxolm74923YamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/crd-nginxolm74923.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmIcspSingleMirrorYaml = []byte(`kind: Template
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

func testQeTestdataOlmIcspSingleMirrorYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmIcspSingleMirrorYaml, nil
}

func testQeTestdataOlmIcspSingleMirrorYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmIcspSingleMirrorYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/icsp-single-mirror.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _testQeTestdataOlmItdmsFullMirrorYaml = []byte(`kind: Template
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

func testQeTestdataOlmItdmsFullMirrorYamlBytes() ([]byte, error) {
	return _testQeTestdataOlmItdmsFullMirrorYaml, nil
}

func testQeTestdataOlmItdmsFullMirrorYaml() (*asset, error) {
	bytes, err := testQeTestdataOlmItdmsFullMirrorYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "test/qe/testdata/olm/itdms-full-mirror.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
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
	"test/qe/testdata/olm/basic-bd-plain-image.yaml":                                                testQeTestdataOlmBasicBdPlainImageYaml,
	"test/qe/testdata/olm/basic-bd-registry-image.yaml":                                             testQeTestdataOlmBasicBdRegistryImageYaml,
	"test/qe/testdata/olm/cip.yaml":                                                                 testQeTestdataOlmCipYaml,
	"test/qe/testdata/olm/clustercatalog-secret-withlabel.yaml":                                     testQeTestdataOlmClustercatalogSecretWithlabelYaml,
	"test/qe/testdata/olm/clustercatalog-secret.yaml":                                               testQeTestdataOlmClustercatalogSecretYaml,
	"test/qe/testdata/olm/clustercatalog-with-pollinterval.yaml":                                    testQeTestdataOlmClustercatalogWithPollintervalYaml,
	"test/qe/testdata/olm/clustercatalog-withlabel.yaml":                                            testQeTestdataOlmClustercatalogWithlabelYaml,
	"test/qe/testdata/olm/clustercatalog.yaml":                                                      testQeTestdataOlmClustercatalogYaml,
	"test/qe/testdata/olm/clusterextension-watchns-config.yaml":                                     testQeTestdataOlmClusterextensionWatchnsConfigYaml,
	"test/qe/testdata/olm/clusterextension-withselectorExpressions-WithoutChannelVersion.yaml":      testQeTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml,
	"test/qe/testdata/olm/clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml": testQeTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml,
	"test/qe/testdata/olm/clusterextension-withselectorlabel-OwnSingle.yaml":                        testQeTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml,
	"test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutChannel.yaml":                   testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml,
	"test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutChannelVersion.yaml":            testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml,
	"test/qe/testdata/olm/clusterextension-withselectorlabel-WithoutVersion.yaml":                   testQeTestdataOlmClusterextensionWithselectorlabelWithoutversionYaml,
	"test/qe/testdata/olm/clusterextension-withselectorlabel-inlineconfig.yaml":                     testQeTestdataOlmClusterextensionWithselectorlabelInlineconfigYaml,
	"test/qe/testdata/olm/clusterextension-withselectorlabel-withoutChannel-OwnSingle.yaml":         testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelOwnsingleYaml,
	"test/qe/testdata/olm/clusterextension-withselectorlabel.yaml":                                  testQeTestdataOlmClusterextensionWithselectorlabelYaml,
	"test/qe/testdata/olm/clusterextension.yaml":                                                    testQeTestdataOlmClusterextensionYaml,
	"test/qe/testdata/olm/clusterextensionWithoutChannel.yaml":                                      testQeTestdataOlmClusterextensionwithoutchannelYaml,
	"test/qe/testdata/olm/clusterextensionWithoutChannelVersion.yaml":                               testQeTestdataOlmClusterextensionwithoutchannelversionYaml,
	"test/qe/testdata/olm/clusterextensionWithoutVersion.yaml":                                      testQeTestdataOlmClusterextensionwithoutversionYaml,
	"test/qe/testdata/olm/cr-webhookTest.yaml":                                                      testQeTestdataOlmCrWebhooktestYaml,
	"test/qe/testdata/olm/crd-nginxolm74923.yaml":                                                   testQeTestdataOlmCrdNginxolm74923Yaml,
	"test/qe/testdata/olm/icsp-single-mirror.yaml":                                                  testQeTestdataOlmIcspSingleMirrorYaml,
	"test/qe/testdata/olm/itdms-full-mirror.yaml":                                                   testQeTestdataOlmItdmsFullMirrorYaml,
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
		"qe": {nil, map[string]*bintree{
			"testdata": {nil, map[string]*bintree{
				"olm": {nil, map[string]*bintree{
					"basic-bd-plain-image.yaml":             {testQeTestdataOlmBasicBdPlainImageYaml, map[string]*bintree{}},
					"basic-bd-registry-image.yaml":          {testQeTestdataOlmBasicBdRegistryImageYaml, map[string]*bintree{}},
					"cip.yaml":                              {testQeTestdataOlmCipYaml, map[string]*bintree{}},
					"clustercatalog-secret-withlabel.yaml":  {testQeTestdataOlmClustercatalogSecretWithlabelYaml, map[string]*bintree{}},
					"clustercatalog-secret.yaml":            {testQeTestdataOlmClustercatalogSecretYaml, map[string]*bintree{}},
					"clustercatalog-with-pollinterval.yaml": {testQeTestdataOlmClustercatalogWithPollintervalYaml, map[string]*bintree{}},
					"clustercatalog-withlabel.yaml":         {testQeTestdataOlmClustercatalogWithlabelYaml, map[string]*bintree{}},
					"clustercatalog.yaml":                   {testQeTestdataOlmClustercatalogYaml, map[string]*bintree{}},
					"clusterextension-watchns-config.yaml":  {testQeTestdataOlmClusterextensionWatchnsConfigYaml, map[string]*bintree{}},
					"clusterextension-withselectorExpressions-WithoutChannelVersion.yaml":      {testQeTestdataOlmClusterextensionWithselectorexpressionsWithoutchannelversionYaml, map[string]*bintree{}},
					"clusterextension-withselectorLableExpressions-WithoutChannelVersion.yaml": {testQeTestdataOlmClusterextensionWithselectorlableexpressionsWithoutchannelversionYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-OwnSingle.yaml":                        {testQeTestdataOlmClusterextensionWithselectorlabelOwnsingleYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-WithoutChannel.yaml":                   {testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-WithoutChannelVersion.yaml":            {testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelversionYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-WithoutVersion.yaml":                   {testQeTestdataOlmClusterextensionWithselectorlabelWithoutversionYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-inlineconfig.yaml":                     {testQeTestdataOlmClusterextensionWithselectorlabelInlineconfigYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel-withoutChannel-OwnSingle.yaml":         {testQeTestdataOlmClusterextensionWithselectorlabelWithoutchannelOwnsingleYaml, map[string]*bintree{}},
					"clusterextension-withselectorlabel.yaml":                                  {testQeTestdataOlmClusterextensionWithselectorlabelYaml, map[string]*bintree{}},
					"clusterextension.yaml":                      {testQeTestdataOlmClusterextensionYaml, map[string]*bintree{}},
					"clusterextensionWithoutChannel.yaml":        {testQeTestdataOlmClusterextensionwithoutchannelYaml, map[string]*bintree{}},
					"clusterextensionWithoutChannelVersion.yaml": {testQeTestdataOlmClusterextensionwithoutchannelversionYaml, map[string]*bintree{}},
					"clusterextensionWithoutVersion.yaml":        {testQeTestdataOlmClusterextensionwithoutversionYaml, map[string]*bintree{}},
					"cr-webhookTest.yaml":                        {testQeTestdataOlmCrWebhooktestYaml, map[string]*bintree{}},
					"crd-nginxolm74923.yaml":                     {testQeTestdataOlmCrdNginxolm74923Yaml, map[string]*bintree{}},
					"icsp-single-mirror.yaml":                    {testQeTestdataOlmIcspSingleMirrorYaml, map[string]*bintree{}},
					"itdms-full-mirror.yaml":                     {testQeTestdataOlmItdmsFullMirrorYaml, map[string]*bintree{}},
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
