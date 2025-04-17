package utils

import (
	"strings"
)

// GetImageNameFrom extracts the image name from the link/url.
func GetImageNameFrom(ref string) string {
	parts := strings.Split(ref, "/")
	last := parts[len(parts)-1]
	if i := strings.Index(last, ":"); i != -1 {
		return last[:i]
	}
	return last
}
