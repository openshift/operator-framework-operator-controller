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
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ocv1 "github.com/operator-framework/operator-controller/api/v1"
)

const (
	ClusterCatalogServingMetricName   = "olm_cluster_catalog_serving"
	ClusterCatalogConditionMetricName = "olm_cluster_catalog_condition"
)

var clusterCatalogConditions = []string{ocv1.TypeServing, ocv1.TypeProgressing}

type clusterCatalogCollector struct {
	reader        client.Reader
	servingDesc   *prometheus.Desc
	conditionDesc *prometheus.Desc
}

func NewClusterCatalogCollector(reader client.Reader) prometheus.Collector {
	return &clusterCatalogCollector{
		reader: reader,
		servingDesc: prometheus.NewDesc(
			ClusterCatalogServingMetricName,
			"Whether a ClusterCatalog is serving content and the digest currently served.",
			[]string{"name", "digest", "reason"},
			nil,
		),
		conditionDesc: prometheus.NewDesc(
			ClusterCatalogConditionMetricName,
			"Current Serving and Progressing conditions for a ClusterCatalog.",
			[]string{"name", "type", "status", "reason"},
			nil,
		),
	}
}

func (c *clusterCatalogCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.servingDesc
	ch <- c.conditionDesc
}

func (c *clusterCatalogCollector) Collect(ch chan<- prometheus.Metric) {
	var catalogs ocv1.ClusterCatalogList
	if err := c.reader.List(context.Background(), &catalogs); err != nil {
		ch <- prometheus.NewInvalidMetric(c.servingDesc, err)
		ch <- prometheus.NewInvalidMetric(c.conditionDesc, err)
		return
	}

	for i := range catalogs.Items {
		catalog := &catalogs.Items[i]
		serving := apimeta.FindStatusCondition(catalog.Status.Conditions, ocv1.TypeServing)
		value, digest, reason := float64(0), "", ""
		if serving != nil {
			reason = serving.Reason
			if serving.Status == metav1.ConditionTrue {
				value = 1
				digest = servedCatalogDigest(catalog)
			}
		}
		ch <- prometheus.MustNewConstMetric(c.servingDesc, prometheus.GaugeValue, value, catalog.Name, digest, reason)

		for _, conditionType := range clusterCatalogConditions {
			condition := apimeta.FindStatusCondition(catalog.Status.Conditions, conditionType)
			if condition == nil {
				continue
			}
			ch <- prometheus.MustNewConstMetric(
				c.conditionDesc,
				prometheus.GaugeValue,
				1,
				catalog.Name,
				condition.Type,
				string(condition.Status),
				condition.Reason,
			)
		}
	}
}

func servedCatalogDigest(catalog *ocv1.ClusterCatalog) string {
	if catalog.Status.ResolvedSource == nil || catalog.Status.ResolvedSource.Image == nil {
		return ""
	}
	_, digest, ok := strings.Cut(catalog.Status.ResolvedSource.Image.Ref, "@")
	if !ok {
		return ""
	}
	return digest
}
