package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"

	apiv1 "github.com/operator-framework/operator-controller/api/v1"
)

// ParseImageRefsFromCatalog reads the catalogs from the files used and returns a list of image references.
func ParseImageRefsFromCatalog(catalogsPath string) ([]string, error) {
	var images []string

	// Check if the directory exists first
	if _, err := os.Stat(catalogsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("catalogs path %s does not exist", catalogsPath)
	}

	err := filepath.WalkDir(catalogsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || (!strings.HasSuffix(d.Name(), ".yaml") &&
			!strings.HasSuffix(d.Name(), ".yml")) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		var catalog apiv1.ClusterCatalog
		if err := yaml.Unmarshal(content, &catalog); err != nil {
			return nil
		}

		if catalog.TypeMeta.Kind != "ClusterCatalog" {
			return nil
		}

		if catalog.Spec.Source.Type == apiv1.SourceTypeImage && catalog.Spec.Source.Image != nil {
			images = append(images, catalog.Spec.Source.Image.Ref)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no images found under catalogs path %s", catalogsPath)
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
