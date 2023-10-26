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

	info := bindataFileInfo{name: "Dockerfile", size: 97, mode: os.FileMode(420), modTime: time.Unix(1760017176, 0)}
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

	info := bindataFileInfo{name: "configs/.indexignore", size: 4, mode: os.FileMode(420), modTime: time.Unix(1760017176, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _configsIndexYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x8c\x52\xb1\x4e\xc3\x30\x14\xdc\xf3\x15\x4f\x9e\x9b\x34\x0c\x2c\xd9\x0a\x64\x40\x82\x14\x91\xd2\x05\x31\x98\xf0\x68\x2c\x12\xc7\xb2\x5d\x43\x65\xf9\xdf\x91\xdd\x44\x55\xa9\xab\xb2\x24\x96\xee\xde\xbb\xf3\x9d\x6d\x02\x40\x54\xd3\x62\x4f\x49\x01\x64\xe8\xfa\x4c\xd0\xe6\x8b\x6e\x90\xcc\x3c\xc4\x69\x8f\x1e\xb0\x16\x56\x65\xbd\x4a\x6f\x5e\xaa\xbb\x87\x12\x9c\x23\x89\x4b\x22\xc3\xef\x5b\xfe\xd1\x5d\x9a\xcd\x4c\x9e\xe5\xd9\xd5\x9e\x35\xc9\x45\x45\x02\x83\xf5\x23\x1e\x0e\xa9\xc4\x0d\x53\x5a\xee\xb2\x41\x20\x57\x2d\xfb\xd4\xe9\x1f\x40\x99\xa6\xb8\xce\xf3\x7c\x6e\x2d\x54\x8b\xc7\xb2\x7e\x5a\xdc\xfa\x75\xf3\x13\x81\xa2\xa3\x1a\x95\x1e\x9d\xc8\x41\xa0\xd4\x0c\x15\x29\xe0\x35\x01\x00\xb0\xe1\x0b\x40\xf4\x4e\x60\x2c\xa0\x00\x1a\xda\x6d\x3d\x3a\xb1\x0f\xb7\xaa\xce\xc6\x37\x3b\x70\x0d\x4a\xc5\x06\xee\x79\xfb\x5c\x46\xc8\x85\xbf\x9b\x9d\x77\xd2\xd3\x9f\xa5\x40\x5e\xfb\x10\xd6\xe3\x96\x13\x57\x5e\x7c\x5d\x3e\xd7\xf7\xcb\x2a\xf4\x36\xad\x7e\x8b\x37\xd8\xb4\x94\x73\xec\x8e\x2b\x14\x12\x0d\xc3\xef\xff\x36\x86\x5c\xcb\x68\x8c\x17\x9f\xc4\x91\xbb\xdf\x00\x00\x00\xff\xff\x62\x45\xa2\xa8\x9d\x02\x00\x00")

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

	info := bindataFileInfo{name: "configs/index.yaml", size: 669, mode: os.FileMode(420), modTime: time.Unix(1760123493, 0)}
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
