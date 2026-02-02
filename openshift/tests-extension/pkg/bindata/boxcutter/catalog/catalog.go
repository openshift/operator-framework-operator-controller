// Code generated for package boxcuttercatalog by go-bindata DO NOT EDIT. (@generated)
// sources:
// testdata/boxcutter/catalog/Dockerfile
// testdata/boxcutter/catalog/configs/index.yaml
package boxcuttercatalog

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

	info := bindataFileInfo{name: "Dockerfile", size: 97, mode: os.FileMode(420), modTime: time.Unix(1763479402, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _configsIndexYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xec\x97\x3b\x6f\xf2\x30\x14\x86\xf7\xfc\x0a\x2b\x33\x84\x7c\xe4\x06\xd9\xf8\x28\x5b\x8b\x2a\x41\xa7\xaa\x83\x09\x07\x88\xea\x5c\x64\x1b\x24\x14\xe5\xbf\x57\xb9\x89\x5e\xdc\x82\xd3\x7a\xf3\x62\x45\x7a\x8f\xfd\x1c\x3f\x52\xa2\x9c\xc2\x40\xc8\x64\xd1\x01\x12\x6c\x86\xc8\xcc\x48\x62\xe5\x38\x7a\xc5\x7b\x30\x07\x55\x94\xe2\x04\xaa\xa0\x28\xd0\x7a\xb1\x5a\x0f\xff\x3f\x2d\xef\xee\x17\xa8\x2c\x9b\x78\x0b\x3b\x7c\x24\x7c\x7e\xc0\x69\x0a\xa4\x2a\x64\x1c\x6f\x08\x98\x46\x69\x08\x8e\xde\x1c\xd3\x2d\xb9\x76\xb2\x75\xfa\x67\xd9\x96\xdd\x54\x75\xcd\x7c\xdf\x42\x9c\xb4\x79\xfd\x30\xa4\xb0\x8f\x19\xa7\x67\x2b\xcb\x21\x65\x87\x78\xc7\x87\x9f\x02\x76\x8a\x42\xcf\xb6\xed\x51\x51\xa0\xe5\xec\x61\xb1\x7a\x9c\xcd\xab\xe3\x46\x5f\x00\x21\xc1\x1c\x18\x6f\x3b\xa1\x59\x0e\x94\xc7\xc0\xcc\x10\x3d\x1b\x08\x21\x54\xd4\x2b\x42\x26\x3f\xe7\x20\xd2\x57\x87\x27\x4c\x8e\x55\xda\x55\x5f\x6e\xb5\xfc\x51\x6e\xb7\x1d\x28\x8b\xb3\xb4\xaa\x6b\xbc\xb4\x51\x69\x74\xeb\xcb\xaf\x64\x8f\xb5\x6c\xa1\xec\xb1\x0a\xd9\x8e\x96\x2d\x94\xed\xa8\x90\xed\x6a\xd9\x42\xd9\xae\x0a\xd9\x9e\x96\x2d\x94\xed\xa9\x90\xed\x6b\xd9\x42\xd9\xbe\x0a\xd9\x81\x96\x2d\x94\x1d\xa8\x90\x3d\xd1\xb2\x85\xb2\x27\x2a\x64\x4f\xb5\x6c\xa1\xec\xa9\x94\xec\xa8\x1d\x7e\x3e\xd8\x6e\xc7\xa0\x1b\xdd\x42\xca\xa9\xf0\xc2\x57\x87\xa4\xa6\xb9\xc1\xcd\xbb\x2e\x7f\xfb\x75\x25\x85\x9c\xe0\xa8\x26\xff\x1d\xc3\x91\x62\x8c\x7b\x31\x5c\x29\x86\xd3\x8b\xe1\x49\x31\xdc\x5e\x0c\x5f\x8a\xe1\xf5\x62\x04\x52\x0c\xbf\x17\x63\x22\xc5\x08\x7a\x31\xa6\x52\x8c\x77\x5f\xcc\xee\xe5\x7d\x0b\x00\x00\xff\xff\x98\x04\x1e\x88\xed\x10\x00\x00")

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

	info := bindataFileInfo{name: "configs/index.yaml", size: 4333, mode: os.FileMode(420), modTime: time.Unix(1763485739, 0)}
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
	"Dockerfile":         dockerfile,
	"configs/index.yaml": configsIndexYaml,
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
		"index.yaml": &bintree{configsIndexYaml, map[string]*bintree{}},
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
