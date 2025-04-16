package catalog

import "github/operator-framework-operator-controller/openshift/catalog-sync/pkg/olmpackage"

var CatalogImages = map[string]string{
	"registry.redhat.io/redhat/community-operator-index:v4.18": "community",
	"registry.redhat.io/redhat/redhat-marketplace-index:v4.18": "marketplace",
	"registry.redhat.io/redhat/certified-operator-index:v4.18": "certified",
	"registry.redhat.io/redhat/redhat-operator-index:v4.18":    "redhat",
}

type Image struct {
	Name     string
	Labels   map[string]string
	Packages []*olmpackage.Package
}
