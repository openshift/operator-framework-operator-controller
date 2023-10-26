// Code generated for package catalog by go-bindata DO NOT EDIT. (@generated)
// sources:
// testdata/catalog/Dockerfile
// testdata/catalog/configs/.indexignore
// testdata/catalog/configs/index.yaml
package catalog

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

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

var _dockerfile = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x72\x0b\xf2\xf7\x55\x28\x4e\x2e\x4a\x2c\x49\xce\xe0\x72\x74\x71\x51\x48\xce\xcf\x4b\xcb\x4c\x2f\x56\xd0\x87\x32\xb8\x7c\x1c\x9d\x5c\x7d\x14\xf2\x0b\x52\x8b\x12\x4b\xf2\x8b\x8a\xf5\x60\xac\xb4\xa2\xc4\xdc\xd4\xf2\xfc\xa2\x6c\xbd\xcc\x7c\xbd\xcc\xbc\x94\xd4\x0a\x3d\xa8\x16\xbd\x32\x43\x5b\xb8\x76\x40\x00\x00\x00\xff\xff\x47\x5b\xe3\x6f\x61\x00\x00\x00")

func dockerfileBytes() ([]byte, error) {
	return bindataRead(
		_dockerfile,
		"Dockerfile",
	)
}

func dockerfile() (*asset, error) {
	bytes, err := dockerfileBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "Dockerfile", size: 97, mode: os.FileMode(420), modTime: time.Unix(1756998653, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _configsIndexignore = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xd2\xd3\xd3\xe2\x02\x04\x00\x00\xff\xff\xcd\x49\xc4\xdc\x04\x00\x00\x00")

func configsIndexignoreBytes() ([]byte, error) {
	return bindataRead(
		_configsIndexignore,
		"configs/.indexignore",
	)
}

func configsIndexignore() (*asset, error) {
	bytes, err := configsIndexignoreBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "configs/.indexignore", size: 4, mode: os.FileMode(420), modTime: time.Unix(1756998653, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _configsIndexYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x84\x91\x31\x4f\xc3\x30\x10\x85\xf7\xfc\x8a\x93\xe7\x24\x0d\x03\x4b\xb6\x02\x19\x90\x20\x45\xa4\x74\x41\x0c\x26\x1c\x8d\x85\xed\x58\xb6\x6b\xa8\xaa\xfc\x77\x64\x27\x55\x69\x71\x61\x49\x2c\x7d\x77\xef\xee\xde\xdb\x25\x00\xc4\xb4\x1d\x0a\x4a\x4a\x20\x3d\x17\xb9\xa2\xed\x07\x5d\x23\x49\x3d\x92\x54\xa0\x07\xcb\xaa\x59\x66\x57\x4f\xf5\xcd\x5d\x45\x92\x21\x89\xb4\xbd\x6e\xe4\x1b\x3f\xdf\x95\xbb\x22\x2f\xf2\x8b\x91\xef\x47\x9c\x08\x07\xc6\xc4\x44\xc2\x23\xd3\xb8\x66\xc6\xea\x6d\xde\x2b\x94\xa6\x63\xef\x36\x3b\x01\xc6\xb5\xe5\x65\x51\x14\xb3\x7a\x7e\x5f\x35\x0f\xf3\xeb\x6a\xf6\x43\xb4\xe4\xd4\xa2\xb1\xd3\x5c\xdd\x2b\xd4\x96\xa1\x21\x25\x3c\x27\x00\x00\xbb\xf0\x05\x20\x76\xab\x30\x66\x41\x80\x8e\xf2\x8d\xa7\xfb\xea\xc3\x0d\x75\xc4\xa0\xf4\x50\xe5\x50\x1b\xd6\x4b\x5f\x31\xde\x3f\xa1\x21\xfc\x87\xf4\xfc\x0e\x82\x7e\x2d\x14\xca\xc6\x9f\xbc\x9a\x54\x7e\xed\x43\x56\xd5\x63\x73\xbb\xa8\x47\x59\x2f\xfa\x12\x4f\xa7\xed\xa8\x94\xc8\x8f\xe3\x51\x1a\x1d\xc3\xcf\xff\x33\x41\x69\x75\xd4\xb4\x3f\x82\x3e\xda\xe8\x3b\x00\x00\xff\xff\x5d\x4e\xb3\xae\x67\x02\x00\x00")

func configsIndexYamlBytes() ([]byte, error) {
	return bindataRead(
		_configsIndexYaml,
		"configs/index.yaml",
	)
}

func configsIndexYaml() (*asset, error) {
	bytes, err := configsIndexYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "configs/index.yaml", size: 615, mode: os.FileMode(420), modTime: time.Unix(1756998653, 0)}
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
	"Dockerfile":           dockerfile,
	"configs/.indexignore": configsIndexignore,
	"configs/index.yaml":   configsIndexYaml,
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
	"Dockerfile": &bintree{dockerfile, map[string]*bintree{}},
	"configs": &bintree{nil, map[string]*bintree{
		".indexignore": &bintree{configsIndexignore, map[string]*bintree{}},
		"index.yaml":   &bintree{configsIndexYaml, map[string]*bintree{}},
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
