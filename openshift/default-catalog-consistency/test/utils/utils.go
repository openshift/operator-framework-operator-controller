package utils

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	yamlutil "k8s.io/apimachinery/pkg/util/yaml"

	apiv1 "github.com/operator-framework/operator-controller/api/v1"
)

// ParseImageRefsFromCatalog reads the catalogs from the files used and returns a list of image references.
func ParseImageRefsFromCatalog(catalogsPath string) ([]string, error) {
	re := regexp.MustCompile(`{{.*}}`)

	info, err := os.Stat(catalogsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("catalogs path %s does not exist", catalogsPath)
		}
		return nil, err
	}

	var (
		images []string
	)

	if info.IsDir() {
		images, err = parseCatalogDir(catalogsPath, re)
	} else {
		images, err = parseCatalogFile(catalogsPath, re)
	}
	if err != nil {
		return nil, err
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no images found under catalogs path %s", catalogsPath)
	}

	return images, nil
}

func parseCatalogDir(root string, re *regexp.Regexp) ([]string, error) {
	var images []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(d.Name(), ".yaml") && !strings.HasSuffix(d.Name(), ".yml") {
			return nil
		}

		refs, err := parseCatalogFile(path, re)
		if err != nil {
			return err
		}

		images = append(images, refs...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return images, nil
}

func parseCatalogFile(path string, re *regexp.Regexp) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseClusterCatalogs(content, re)
}

func parseClusterCatalogs(content []byte, re *regexp.Regexp) ([]string, error) {
	// Replace any helm templating
	content = re.ReplaceAll(content, []byte{})

	decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(content), 4096)

	var images []string
	for {
		var catalog apiv1.ClusterCatalog
		if err := decoder.Decode(&catalog); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		if catalog.Kind != "ClusterCatalog" {
			continue
		}

		if catalog.Spec.Source.Type == apiv1.SourceTypeImage && catalog.Spec.Source.Image != nil {
			images = append(images, catalog.Spec.Source.Image.Ref)
		}
	}

	return images, nil
}

// ImageNameFromRef extracts the image name from the link/url.
func ImageNameFromRef(ref string) string {
	parts := strings.Split(ref, "/")
	last := parts[len(parts)-1]
	if i := strings.Index(last, ":"); i != -1 {
		return last[:i]
	}
	return last
}
