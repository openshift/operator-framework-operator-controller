/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"context"
	"sort"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"
)

const (
	ClusterExtensionInfoMetricName      = "olm_clusterextension_info"
	ClusterExtensionConditionMetricName = "olm_clusterextension_condition"
)

var (
	clusterExtensionInfoLabels = []string{"name", "package", "channels", "catalog", "installed_version"}
	clusterExtensionConditions = []string{ocv1.TypeInstalled, ocv1.TypeProgressing}
	conditionLabels            = []string{"name", "type", "status", "reason"}
)

type clusterExtensionCollector struct {
	reader        client.Reader
	infoDesc      *prometheus.Desc
	conditionDesc *prometheus.Desc
}

func NewClusterExtensionCollector(reader client.Reader) prometheus.Collector {
	return &clusterExtensionCollector{
		reader: reader,
		infoDesc: prometheus.NewDesc(
			ClusterExtensionInfoMetricName,
			"Information about a ClusterExtension and its current installation.",
			clusterExtensionInfoLabels,
			nil,
		),
		conditionDesc: prometheus.NewDesc(
			ClusterExtensionConditionMetricName,
			"Current Installed and Progressing conditions for a ClusterExtension.",
			append([]string{"package"}, conditionLabels...),
			nil,
		),
	}
}

func (c *clusterExtensionCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc
	ch <- c.conditionDesc
}

func (c *clusterExtensionCollector) Collect(ch chan<- prometheus.Metric) {
	var extensions ocv1.ClusterExtensionList
	if err := c.reader.List(context.Background(), &extensions); err != nil {
		ch <- prometheus.NewInvalidMetric(c.infoDesc, err)
		ch <- prometheus.NewInvalidMetric(c.conditionDesc, err)
		return
	}

	for i := range extensions.Items {
		extension := &extensions.Items[i]
		packageName, channels, catalogName := extensionSourceValues(extension)
		installedVersion := ""
		if extension.Status.Install != nil {
			installedVersion = extension.Status.Install.Bundle.Version
		}
		ch <- prometheus.MustNewConstMetric(
			c.infoDesc,
			prometheus.GaugeValue,
			1,
			extension.Name,
			packageName,
			channels,
			catalogName,
			installedVersion,
		)
		collectConditions(ch, c.conditionDesc, extension.Status.Conditions, clusterExtensionConditions, packageName, extension.Name)
	}
}

func extensionSourceValues(extension *ocv1.ClusterExtension) (packageName, channels, catalogName string) {
	catalog := extension.Spec.Source.Catalog
	if catalog == nil {
		return "", "", ""
	}
	requestedChannels := append([]string(nil), catalog.Channels...)
	sort.Strings(requestedChannels)
	if catalog.Selector != nil {
		catalogName = catalog.Selector.MatchLabels[ocv1.MetadataNameLabel]
	}
	return catalog.PackageName, strings.Join(requestedChannels, ","), catalogName
}

func collectConditions(ch chan<- prometheus.Metric, desc *prometheus.Desc, conditions []metav1.Condition, conditionTypes []string, labels ...string) {
	for _, conditionType := range conditionTypes {
		condition := apimeta.FindStatusCondition(conditions, conditionType)
		if condition == nil {
			continue
		}
		ch <- prometheus.MustNewConstMetric(
			desc,
			prometheus.GaugeValue,
			1,
			append(labels, condition.Type, string(condition.Status), condition.Reason)...,
		)
	}
}
