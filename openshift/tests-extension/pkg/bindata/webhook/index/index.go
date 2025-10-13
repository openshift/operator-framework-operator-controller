// Code generated for package webhookindex by go-bindata DO NOT EDIT. (@generated)
// sources:
// testdata/webhook/index/Dockerfile
// testdata/webhook/index/configs/index.yaml
package webhookindex

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

	info := bindataFileInfo{name: "Dockerfile", size: 97, mode: os.FileMode(420), modTime: time.Unix(1760370604, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _configsIndexYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xd4\x53\x3d\x6f\x83\x30\x10\xdd\xf9\x15\x96\xe7\xe0\xd2\x4a\x59\xd8\xd2\x34\x5b\x1b\x55\x4a\xaa\x0e\x55\x07\x07\x2e\x60\x61\x6c\xcb\x36\x54\x11\xe2\xbf\x57\xe6\x2b\x05\x35\x24\x52\xa7\x66\xba\xdc\x7b\xef\xee\x3d\xa3\xab\x3c\x84\x10\xc2\x26\x4a\x21\xa7\x38\x44\x58\xf2\x9c\x28\x1a\x65\x34\x01\xbc\x68\x41\x41\x73\x70\xd0\x17\x1c\x52\x29\x33\x5f\x2a\xd0\xd4\x4a\xdd\xe3\x31\x1c\x69\xc1\xed\x3a\xa5\x42\x00\x77\x4c\xca\x55\x4a\xb1\x57\x7b\xbf\x8e\x8f\x3a\xe2\x64\x7c\x2b\xea\x9a\xbd\x85\x99\xb5\x20\xac\x66\x60\x70\x88\x3e\x9a\x86\xfb\x55\x43\x35\xeb\x9c\x94\x01\x09\xc8\x12\x0f\xec\xba\xa9\x3e\x2f\x39\x3e\x14\x22\xe6\x57\xdf\xa3\x9f\x7a\x7b\x04\x96\x77\x84\xa6\xf0\x35\x24\xcc\x58\x7d\x22\x52\x81\x30\x29\x3b\x5a\x7f\x02\x98\x32\x0a\x97\x41\x10\xdc\x55\x15\xda\xae\x5e\x36\xbb\xd7\xd5\x7a\x83\xea\xda\xfd\xdf\x6f\x76\x7b\xff\xf1\x6d\xfb\xf4\xec\x3a\x21\xa7\x16\x8c\x1d\xcc\x68\xb7\xdb\x5e\x79\x2f\x7b\x52\xd0\x67\x4e\xca\xac\x13\x0f\x70\x49\x79\xe1\xf0\xb1\xaa\x81\x12\x2d\x0b\xf5\x23\x2a\xe9\xa3\x1a\x12\x49\x0d\xd2\x10\x26\x27\xe3\x1a\x5d\xc6\x44\xec\x64\xef\xad\x6c\x7f\xb6\x3c\xde\x0c\xda\x30\x29\x1c\xb3\xbc\xc7\x23\xbc\x3e\x7f\xc4\xc5\xbf\xce\xf5\xf0\xb7\x5c\xe3\xa3\xbd\x25\x5b\xa7\xd8\xce\x9f\xf7\x25\xbf\x93\x0b\x9a\x58\x1e\xee\xc9\xfb\x0e\x00\x00\xff\xff\x10\x43\x2d\xf5\x62\x04\x00\x00")

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

	info := bindataFileInfo{name: "configs/index.yaml", size: 1122, mode: os.FileMode(420), modTime: time.Unix(1760370604, 0)}
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
