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

var _configsIndexYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x92\x31\x4f\xc3\x30\x10\x85\xf7\xfc\x8a\x93\xe7\x26\x0d\x03\x4b\xb6\x02\x19\x90\x20\x45\xa4\x74\x41\x0c\x26\x1c\x8d\x45\xe2\x58\xb6\x6b\xa8\x2c\xff\x77\x64\x27\x55\x55\xea\xaa\x2c\x49\xa4\xef\xf9\xdd\xcb\x3b\xdb\x04\x80\xa8\xa6\xc5\x9e\x92\x02\xc8\xd0\xf5\x99\xa0\xcd\x17\xdd\x20\x99\x79\xc4\x69\x8f\x1e\x58\x0b\xab\xb2\x5e\xa5\x37\x2f\xd5\xdd\x43\x09\xce\x91\xc4\x25\x91\xc3\xef\x5b\xfe\xd1\x5d\x3a\x9b\x99\x3c\xcb\xb3\xab\x51\xb5\x1f\x17\x1d\x12\x14\xac\x9f\x78\xf8\x48\x25\x6e\x98\xd2\x72\x97\x0d\x02\xb9\x6a\xd9\xa7\x4e\xff\x00\x65\x9a\xe2\x3a\xcf\xf3\xb9\xb5\x50\x2d\x1e\xcb\xfa\x69\x71\xeb\xed\xe6\x27\x03\x8a\x8e\x6a\x54\x7a\x4a\x22\x07\x81\x52\x33\x54\xa4\x80\xd7\x04\x00\xc0\x86\x27\x00\xd1\x3b\x81\xb1\x82\x02\x34\xb4\xdb\x7a\xba\x57\x1f\xfe\xaa\x3a\x5b\xdf\xec\xa0\x35\x28\x15\x1b\xb8\xd7\x8d\xbd\x4c\xc8\x85\xb7\x9b\x9d\x4f\xd2\xd3\x9f\xa5\x40\x5e\xfb\x12\xd6\x93\xcb\x49\x2a\xb2\x2e\x9f\xeb\xfb\x65\x35\xda\x7a\xd3\xb7\xf8\xee\x9a\x96\x72\x8e\xdd\xf1\xf2\x84\x44\xc3\xf0\xfb\xbf\xbb\x42\xae\x65\xb4\xc0\x8b\x97\xe1\x28\xdd\x6f\x00\x00\x00\xff\xff\xe5\x9f\x1d\x23\x97\x02\x00\x00")

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

	info := bindataFileInfo{name: "configs/index.yaml", size: 663, mode: os.FileMode(420), modTime: time.Unix(1759218366, 0)}
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
