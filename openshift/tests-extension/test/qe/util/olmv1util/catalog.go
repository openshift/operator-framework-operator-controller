package olmv1util

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/wait"
	e2e "k8s.io/kubernetes/test/e2e/framework"

	exutil "github.com/openshift/operator-framework-operator-controller/openshift/tests-extension/test/qe/util"
)

const (
	v1ApiPath = "api/v1"
	v1ApiData = "all"
)

type ClusterCatalogDescription struct {
	Name                string
	PullSecret          string
	TypeName            string
	Imageref            string
	ContentURL          string
	Status              string
	PollIntervalMinutes string
	LabelKey            string // default is olmv1-test
	LabelValue          string // suggest to use case id
	Template            string
}

// Create creates a cluster catalog and waits for it to reach serving status
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (clustercatalog *ClusterCatalogDescription) Create(oc *exutil.CLI) {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	o.Expect(clustercatalog.Name).NotTo(o.BeEmpty(), "cluster catalog name cannot be empty")

	e2e.Logf("=========Create clustercatalog %v=========", clustercatalog.Name)
	err := clustercatalog.CreateWithoutCheck(oc)
	o.Expect(err).NotTo(o.HaveOccurred())
	clustercatalog.WaitCatalogStatus(oc, "true", "Serving", 0)
	clustercatalog.GetContentURL(oc)
}

// CreateWithoutCheck creates a cluster catalog from template without waiting for status verification
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - error: error if template application fails, nil on success
func (clustercatalog *ClusterCatalogDescription) CreateWithoutCheck(oc *exutil.CLI) error {
	if oc == nil {
		return fmt.Errorf("CLI client cannot be nil")
	}
	if clustercatalog.Template == "" {
		return fmt.Errorf("template path cannot be empty")
	}

	parameters := []string{"--ignore-unknown-parameters=true", "-f", clustercatalog.Template, "-p"}
	if len(oc.Namespace()) == 0 {
		parameters = []string{"-n", "default", "--ignore-unknown-parameters=true", "-f", clustercatalog.Template, "-p"}
	}
	if len(clustercatalog.Name) > 0 {
		parameters = append(parameters, "NAME="+clustercatalog.Name)
	}
	if len(clustercatalog.PullSecret) > 0 {
		parameters = append(parameters, "SECRET="+clustercatalog.PullSecret)
	}
	if len(clustercatalog.TypeName) > 0 {
		parameters = append(parameters, "TYPE="+clustercatalog.TypeName)
	}
	if len(clustercatalog.Imageref) > 0 {
		parameters = append(parameters, "IMAGE="+clustercatalog.Imageref)
	}
	if len(clustercatalog.PollIntervalMinutes) > 0 {
		parameters = append(parameters, "POLLINTERVALMINUTES="+clustercatalog.PollIntervalMinutes)
	}
	if len(clustercatalog.LabelKey) > 0 {
		parameters = append(parameters, "LABELKEY="+clustercatalog.LabelKey)
	}
	if len(clustercatalog.LabelValue) > 0 {
		parameters = append(parameters, "LABELVALUE="+clustercatalog.LabelValue)
	}
	err := exutil.ApplyClusterResourceFromTemplateWithError(oc, parameters...)
	return err
}

// WaitCatalogStatus waits for the cluster catalog to reach a specific status condition
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - status: expected status value (e.g., "true", "false")
//   - conditionType: type of condition to check (e.g., "Serving", "Ready")
//   - consistentTime: time in seconds to verify status remains consistent (0 to skip consistency check)
func (clustercatalog *ClusterCatalogDescription) WaitCatalogStatus(oc *exutil.CLI, status string, conditionType string, consistentTime int) {
	e2e.Logf("========= check clustercatalog %v status is %s =========", clustercatalog.Name, status)

	jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].status}`, conditionType)
	errWait := wait.PollUntilContextTimeout(context.TODO(), 3*time.Second, 150*time.Second, false, func(ctx context.Context) (bool, error) {
		output, err := GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o", jsonpath)
		if err != nil {
			e2e.Logf("output is %v, error is %v, and try next", output, err)
			return false, nil
		}
		if !strings.Contains(strings.ToLower(output), strings.ToLower(status)) {
			e2e.Logf("status is %v, not %v, and try next", output, status)
			clustercatalog.Status = output
			return false, nil
		}
		return true, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster catalog debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster catalog debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("clustercatalog status is not %s", status))
	}
	if consistentTime != 0 {
		e2e.Logf("make sure clustercatalog %s status is %s consistently for %ds", clustercatalog.Name, status, consistentTime)
		o.Consistently(func() string {
			output, _ := GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o", jsonpath)
			return strings.ToLower(output)
		}, time.Duration(consistentTime)*time.Second, 5*time.Second).Should(o.ContainSubstring(strings.ToLower(status)),
			"clustercatalog %s status is not %s", clustercatalog.Name, status)
	}
}

// CheckClusterCatalogCondition checks a specific field within a condition type of the cluster catalog
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - conditionType: type of condition to check (e.g., "Serving", "Ready")
//   - field: specific field within the condition (e.g., "status", "reason", "message")
//   - expect: expected value for the field
//   - checkInterval: interval between checks in seconds
//   - checkTimeout: maximum time to wait in seconds
//   - consistentTime: time in seconds to verify value remains consistent (0 to skip consistency check)
func (clustercatalog *ClusterCatalogDescription) CheckClusterCatalogCondition(oc *exutil.CLI, conditionType, field, expect string, checkInterval, checkTimeout, consistentTime int) {
	e2e.Logf("========= check clustercatalog %v %s %s expect is %s =========", clustercatalog.Name, conditionType, field, expect)
	jsonpath := fmt.Sprintf(`jsonpath={.status.conditions[?(@.type=="%s")].%s}`, conditionType, field)
	errWait := wait.PollUntilContextTimeout(context.TODO(), time.Duration(checkInterval)*time.Second, time.Duration(checkTimeout)*time.Second, false, func(ctx context.Context) (bool, error) {
		output, err := GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o", jsonpath)
		if err != nil {
			e2e.Logf("output is %v, error is %v, and try next", output, err)
			return false, nil
		}
		if !strings.Contains(strings.ToLower(output), strings.ToLower(expect)) {
			e2e.Logf("got is %v, not %v, and try next", output, expect)
			return false, nil
		}
		return true, nil
	})
	if errWait != nil {
		if debugOutput, debugErr := GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o=jsonpath-as-json={.status}"); debugErr != nil {
			e2e.Logf("Failed to get cluster catalog debug info: %v", debugErr)
		} else {
			e2e.Logf("Cluster catalog debug status: %s", debugOutput)
		}
		exutil.AssertWaitPollNoErr(errWait, fmt.Sprintf("clustercatalog %s expected is not %s in %v seconds", conditionType, expect, checkTimeout))
	}
	if consistentTime != 0 {
		e2e.Logf("make sure clustercatalog %s expect is %s consistently for %ds", conditionType, expect, consistentTime)
		o.Consistently(func() string {
			output, _ := GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o", jsonpath)
			return strings.ToLower(output)
		}, time.Duration(consistentTime)*time.Second, 5*time.Second).Should(o.ContainSubstring(strings.ToLower(expect)),
			"clustercatalog %s expected is not %s", conditionType, expect)
	}
}

// GetContentURL retrieves and sets the content URL for accessing the cluster catalog's API endpoint
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (clustercatalog *ClusterCatalogDescription) GetContentURL(oc *exutil.CLI) {
	e2e.Logf("=========Get clustercatalog %v contentURL =========", clustercatalog.Name)
	route, err := oc.AsAdmin().WithoutNamespace().Run("get").Args("route", "catalogd-service", "-n", "openshift-catalogd", "-o=jsonpath={.spec.host}").Output()
	if err != nil && !strings.Contains(route, "NotFound") {
		o.Expect(err).NotTo(o.HaveOccurred())
	}
	if route == "" || err != nil {
		output, err := oc.AsAdmin().WithoutNamespace().Run("create").Args("route", "reencrypt", "--service=catalogd-service", "-n", "openshift-catalogd").Output()
		e2e.Logf("output is %v, error is %v", output, err)
		errWait := wait.PollUntilContextTimeout(context.TODO(), 2*time.Second, 10*time.Second, false, func(ctx context.Context) (bool, error) {
			route, err = oc.AsAdmin().WithoutNamespace().Run("get").Args("route", "catalogd-service", "-n", "openshift-catalogd", "-o=jsonpath={.spec.host}").Output()
			if err != nil {
				e2e.Logf("output is %v, error is %v, and try next", route, err)
				return false, nil
			}
			if route == "" {
				e2e.Logf("route is empty")
				return false, nil
			}
			return true, nil
		})
		exutil.AssertWaitPollNoErr(errWait, "get route catalogd-service failed")
	}
	o.Expect(route).To(o.ContainSubstring("catalogd-service-openshift-catalogd"))
	contentURL, err := GetNoEmpty(oc, "clustercatalog", clustercatalog.Name, "-o", "jsonpath={.status.urls.base}")
	o.Expect(err).NotTo(o.HaveOccurred())
	contentURL = contentURL + "/" + v1ApiPath + "/" + v1ApiData
	clustercatalog.ContentURL = strings.Replace(contentURL, "catalogd-service.openshift-catalogd.svc", route, 1)
	e2e.Logf("clustercatalog contentURL is %s", clustercatalog.ContentURL)
}

// DeleteWithoutCheck deletes the cluster catalog resource without verification
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (clustercatalog *ClusterCatalogDescription) DeleteWithoutCheck(oc *exutil.CLI) {
	e2e.Logf("=========DeleteWithoutCheck clustercatalog %v=========", clustercatalog.Name)
	exutil.CleanupResource(oc, 4*time.Second, 160*time.Second, exutil.AsAdmin, exutil.WithoutNamespace, "clustercatalog", clustercatalog.Name)
}

// Delete removes the cluster catalog and performs cleanup operations
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
func (clustercatalog *ClusterCatalogDescription) Delete(oc *exutil.CLI) {
	e2e.Logf("=========Delete clustercatalog %v=========", clustercatalog.Name)
	clustercatalog.DeleteWithoutCheck(oc)
	//add check later
}

// Patch applies a merge patch to the cluster catalog resource
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - patch: JSON patch string to apply to the resource
func (clustercatalog *ClusterCatalogDescription) Patch(oc *exutil.CLI, patch string) {
	_, err := oc.AsAdmin().WithoutNamespace().Run("patch").Args("clustercatalog", clustercatalog.Name, "--type", "merge", "-p", patch).Output()
	o.Expect(err).NotTo(o.HaveOccurred())
}

// GetContent retrieves the raw content data from the cluster catalog's API endpoint
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - []byte: raw catalog content data from the API endpoint
func (clustercatalog *ClusterCatalogDescription) GetContent(oc *exutil.CLI) []byte {
	o.Expect(oc).NotTo(o.BeNil(), "CLI client cannot be nil")
	if clustercatalog.ContentURL == "" {
		clustercatalog.GetContentURL(oc)
	}

	var proxy string
	if os.Getenv("http_proxy") != "" {
		proxy = os.Getenv("http_proxy")
	} else if os.Getenv("https_proxy") != "" {
		proxy = os.Getenv("https_proxy")
	} else if os.Getenv("HTTP_PROXY") != "" {
		proxy = os.Getenv("HTTP_PROXY")
	} else if os.Getenv("HTTPS_PROXY") != "" {
		proxy = os.Getenv("HTTPS_PROXY")
	}

	var tr *http.Transport
	if len(proxy) > 0 {
		e2e.Logf("take proxy to access cluster")
		proxyURL, err := url.Parse(proxy)
		o.Expect(err).NotTo(o.HaveOccurred())
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{
				// Warning: InsecureSkipVerify is used for test environments only
				// In production, proper certificate validation should be implemented
				InsecureSkipVerify: true, // nolint:gosec // G402: InsecureSkipVerify is acceptable in test environment
			},
			Proxy: http.ProxyURL(proxyURL),
		}
	} else {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{
				// Warning: InsecureSkipVerify is used for test environments only
				// In production, proper certificate validation should be implemented
				InsecureSkipVerify: true, // nolint:gosec // G402: InsecureSkipVerify is acceptable in test environment
			},
		}
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Get(clustercatalog.ContentURL)
	if err != nil && strings.Contains(err.Error(), "Service Unavailable") {
		g.Skip("the service can not be accessible with Service Unavailable")
	}
	o.Expect(err).NotTo(o.HaveOccurred())

	defer func() {
		if resp != nil && resp.Body != nil {
			if closeErr := resp.Body.Close(); closeErr != nil {
				e2e.Logf("Error closing response body: %v", closeErr)
			}
		}
	}()
	curlOutput, err := io.ReadAll(resp.Body)
	o.Expect(err).NotTo(o.HaveOccurred())
	return curlOutput
}

// RelatedImagesInfo returns the relatedImages info
type RelatedImagesInfo struct {
	Image string `json:"image"`
	Name  string `json:"name"`
}

// BundleData returns the bundle info
type BundleData struct {
	Image         string              `json:"image"`
	Name          string              `json:"name"`
	Package       string              `json:"package"`
	RelatedImages []RelatedImagesInfo `json:"relatedImages"`
	Schema        string              `json:"schema"`
	Properties    json.RawMessage     `json:"properties"` // properties data are complex and will be output as strings
}

// GetBundlesName extracts bundle names from a slice of BundleData
// Parameters:
//   - bundlesDataOut: slice of BundleData structs to extract names from
//
// Returns:
//   - []string: slice of bundle names
func GetBundlesName(bundlesDataOut []BundleData) []string {
	bundlesName := make([]string, 0, len(bundlesDataOut))
	var singleBundleData BundleData

	for _, singleBundleData = range bundlesDataOut {
		bundlesName = append(bundlesName, singleBundleData.Name)
	}
	return bundlesName
}

// GetBundlesNameByPakcage extracts bundle names that belong to a specific package
// Parameters:
//   - bundlesDataOut: slice of BundleData structs to search through
//   - packageName: name of the package to filter bundles by
//
// Returns:
//   - []string: slice of bundle names belonging to the specified package
func GetBundlesNameByPakcage(bundlesDataOut []BundleData, packageName string) []string {
	var bundlesName []string
	var singleBundleData BundleData

	for _, singleBundleData = range bundlesDataOut {
		if singleBundleData.Package == packageName {
			bundlesName = append(bundlesName, singleBundleData.Name)
		}
	}
	return bundlesName
}

// GetBundlesImageTag extracts image tags from a slice of BundleData
// Parameters:
//   - bundlesDataOut: slice of BundleData structs to extract image tags from
//
// Returns:
//   - []string: slice of bundle image tags
func GetBundlesImageTag(bundlesDataOut []BundleData) []string {
	bundlesName := make([]string, 0, len(bundlesDataOut))
	var singleBundleData BundleData

	for _, singleBundleData = range bundlesDataOut {
		bundlesName = append(bundlesName, singleBundleData.Image)
	}
	return bundlesName
}

// GetBundleInfoByName finds and returns bundle information by package and bundle name
// Parameters:
//   - bundlesDataOut: slice of BundleData structs to search through
//   - packageName: name of the package the bundle belongs to
//   - bundleName: name of the bundle to find
//
// Returns:
//   - *BundleData: pointer to matching BundleData struct, nil if not found
func GetBundleInfoByName(bundlesDataOut []BundleData, packageName string, bundleName string) *BundleData {
	var singleBundleData BundleData

	for _, singleBundleData = range bundlesDataOut {
		if singleBundleData.Name == bundleName && singleBundleData.Package == packageName {
			return &singleBundleData
		}
	}
	return nil
}

// EntriesInfo returns the entries info
type EntriesInfo struct {
	Name     string   `json:"name"`
	Replaces string   `json:"replaces"`
	Skips    []string `json:"skips"`
}

// ChannelData returns the channel info
type ChannelData struct {
	Entries []EntriesInfo `json:"entries"`
	Name    string        `json:"name"`
	Package string        `json:"package"`
	Schema  string        `json:"schema"`
}

// GetChannelByPakcage retrieves all channel data for a specific package
// Parameters:
//   - channelDataOut: slice of ChannelData structs to search through
//   - packageName: name of the package to filter channels by
//
// Returns:
//   - []ChannelData: slice of ChannelData structs belonging to the specified package
func GetChannelByPakcage(channelDataOut []ChannelData, packageName string) []ChannelData {
	var channelDataByPackage []ChannelData
	var singleChannelData ChannelData
	for _, singleChannelData = range channelDataOut {
		if singleChannelData.Package == packageName {
			channelDataByPackage = append(channelDataByPackage, singleChannelData)
		}
	}
	return channelDataByPackage
}

// GetChannelNameByPakcage extracts channel names that belong to a specific package
// Parameters:
//   - channelDataOut: slice of ChannelData structs to search through
//   - packageName: name of the package to filter channels by
//
// Returns:
//   - []string: slice of channel names belonging to the specified package
func GetChannelNameByPakcage(channelDataOut []ChannelData, packageName string) []string {
	var channelsName []string
	var singleChannelData ChannelData

	for _, singleChannelData = range channelDataOut {
		if singleChannelData.Package == packageName {
			channelsName = append(channelsName, singleChannelData.Name)
		}
	}
	return channelsName
}

// PackageData returns the package info
type PackageData struct {
	DefaultChannel string `json:"defaultChannel"`
	Name           string `json:"name"`
	Schema         string `json:"schema"`
}

// ListPackagesName extracts package names from a slice of PackageData
// Parameters:
//   - packageDataOut: slice of PackageData structs to extract names from
//
// Returns:
//   - []string: slice of package names
func ListPackagesName(packageDataOut []PackageData) []string {
	packagesName := make([]string, 0, len(packageDataOut))
	var singlePackageData PackageData

	for _, singlePackageData = range packageDataOut {
		packagesName = append(packagesName, singlePackageData.Name)
	}
	return packagesName
}

// ReferenceInfo returns the Reference info
type ReferenceInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

// EntriesInfo returns the entries info
type DeprecatedEntriesInfo struct {
	Message   string        `json:"message"`
	Reference ReferenceInfo `json:"reference"`
}

// DeprecationData returns the deprecated info
type DeprecationData struct {
	Entries []DeprecatedEntriesInfo `json:"entries"`
	Package string                  `json:"package"`
	Schema  string                  `json:"schema"`
}

// GetDeprecatedChannelNameByPakcage extracts deprecated channel names for a specific package
// Parameters:
//   - deprecationDataOut: slice of DeprecationData structs to search through
//   - packageName: name of the package to find deprecated channels for
//
// Returns:
//   - []string: slice of deprecated channel names for the specified package
func GetDeprecatedChannelNameByPakcage(deprecationDataOut []DeprecationData, packageName string) []string {
	var channelsName []string
	var singleDeprecationData DeprecationData
	var deprecatedEntriesInfo DeprecatedEntriesInfo

	for _, singleDeprecationData = range deprecationDataOut {
		if singleDeprecationData.Package == packageName {
			for _, deprecatedEntriesInfo = range singleDeprecationData.Entries {
				if deprecatedEntriesInfo.Reference.Schema == "olm.channel" {
					channelsName = append(channelsName, deprecatedEntriesInfo.Reference.Name)
				}
			}
		}
	}
	return channelsName
}

// GetDeprecatedBundlesNameByPakcage extracts deprecated bundle names for a specific package
// Parameters:
//   - deprecationDataOut: slice of DeprecationData structs to search through
//   - packageName: name of the package to find deprecated bundles for
//
// Returns:
//   - []string: slice of deprecated bundle names for the specified package
func GetDeprecatedBundlesNameByPakcage(deprecationDataOut []DeprecationData, packageName string) []string {
	var bundlesName []string
	var singleDeprecationData DeprecationData
	var deprecatedEntriesInfo DeprecatedEntriesInfo

	for _, singleDeprecationData = range deprecationDataOut {
		if singleDeprecationData.Package == packageName {
			for _, deprecatedEntriesInfo = range singleDeprecationData.Entries {
				if deprecatedEntriesInfo.Reference.Schema == "olm.bundle" {
					bundlesName = append(bundlesName, deprecatedEntriesInfo.Reference.Name)
				}
			}
		}
	}
	return bundlesName
}

type ContentData struct {
	Packages     []PackageData
	Channels     []ChannelData
	Bundles      []BundleData
	Deprecations []DeprecationData
}

// UnmarshalContent parses catalog content for a specific schema type (bundle, channel, package, deprecations, or all)
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//   - schema: type of content to unmarshal ("bundle", "channel", "package", "deprecations", or "all")
//
// Returns:
//   - ContentData: parsed content data containing the requested schema information
//   - error: error if content retrieval or parsing fails, nil on success
func (clustercatalog *ClusterCatalogDescription) UnmarshalContent(oc *exutil.CLI, schema string) (ContentData, error) {
	var (
		singlePackageData     PackageData
		singleChannelData     ChannelData
		singleBundleData      BundleData
		singleDeprecationData DeprecationData
		ContentData           ContentData
		targetData            interface{}
		err                   error
	)

	switch schema {
	case "all":
		return clustercatalog.UnmarshalAllContent(oc)
	case "bundle":
		targetData = &singleBundleData
	case "channel":
		targetData = &singleChannelData
	case "package":
		targetData = &singlePackageData
	case "deprecations":
		targetData = &singleDeprecationData
	default:
		return ContentData, fmt.Errorf("unsupported schema: %s", schema)
	}

	contents := clustercatalog.GetContent(oc)
	lines := strings.Split(string(contents), "\n")

	for _, line := range lines {
		if strings.Contains(line, "\"schema\":\"olm."+schema+"\"") {
			if err = json.Unmarshal([]byte(line), targetData); err != nil {
				return ContentData, err
			}

			switch schema {
			case "bundle":
				ContentData.Bundles = append(ContentData.Bundles, singleBundleData)
			case "channel":
				ContentData.Channels = append(ContentData.Channels, singleChannelData)
			case "package":
				ContentData.Packages = append(ContentData.Packages, singlePackageData)
			case "deprecations":
				ContentData.Deprecations = append(ContentData.Deprecations, singleDeprecationData)
			}
		}
	}

	err = nil

	switch schema {
	case "bundle":
		if len(ContentData.Bundles) == 0 {
			err = fmt.Errorf("can not get Bundles")
		}
	case "channel":
		if len(ContentData.Channels) == 0 {
			err = fmt.Errorf("can not get Channels")
		}
	case "package":
		if len(ContentData.Packages) == 0 {
			err = fmt.Errorf("can not get Packages")
		}
	case "deprecations":
		if len(ContentData.Deprecations) == 0 {
			err = fmt.Errorf("can not get Deprecations")
		}
	}
	return ContentData, err
}

// UnmarshalAllContent parses all catalog content including bundles, channels, packages, and deprecations
// Parameters:
//   - oc: CLI client for interacting with the OpenShift cluster
//
// Returns:
//   - ContentData: parsed content data containing all schema types
//   - error: error if content retrieval or parsing fails, nil on success
func (clustercatalog *ClusterCatalogDescription) UnmarshalAllContent(oc *exutil.CLI) (ContentData, error) {
	var ContentData ContentData

	contents := clustercatalog.GetContent(oc)
	lines := strings.Split(string(contents), "\n")

	for _, line := range lines {
		if strings.Contains(line, "\"schema\":\"olm.bundle\"") || strings.Contains(line, "\"schema\":\"olm.channel\"") || strings.Contains(line, "\"schema\":\"olm.package\"") || strings.Contains(line, "\"schema\":\"olm.deprecations\"") {
			var targetData interface{}
			switch {
			case strings.Contains(line, "\"schema\":\"olm.bundle\""):
				targetData = new(BundleData)
			case strings.Contains(line, "\"schema\":\"olm.channel\""):
				targetData = new(ChannelData)
			case strings.Contains(line, "\"schema\":\"olm.package\""):
				targetData = new(PackageData)
			case strings.Contains(line, "\"schema\":\"olm.deprecations\""):
				targetData = new(DeprecationData)
			}

			if err := json.Unmarshal([]byte(line), targetData); err != nil {
				return ContentData, err
			}

			switch data := targetData.(type) {
			case *BundleData:
				ContentData.Bundles = append(ContentData.Bundles, *data)
			case *ChannelData:
				ContentData.Channels = append(ContentData.Channels, *data)
			case *PackageData:
				ContentData.Packages = append(ContentData.Packages, *data)
			case *DeprecationData:
				ContentData.Deprecations = append(ContentData.Deprecations, *data)
			}
		}
	}
	if len(ContentData.Bundles) == 0 && len(ContentData.Channels) == 0 && len(ContentData.Packages) == 0 && len(ContentData.Deprecations) == 0 {
		return ContentData, fmt.Errorf("no any bundle, channel or package are got")
	}
	return ContentData, nil
}
