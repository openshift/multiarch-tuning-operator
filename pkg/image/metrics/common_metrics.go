package metrics

import (
	"sync"

	"github.com/openshift/multiarch-tuning-operator/pkg/utils"

	"github.com/prometheus/client_golang/prometheus"
	metrics2 "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var onceCommon sync.Once

var (
	InspectionGauge             prometheus.Gauge
	TimeToInspectImageGivenHit  prometheus.Histogram
	TimeToInspectImageGivenMiss prometheus.Histogram
)

func InitCommonMetrics() {
	onceCommon.Do(func() {
		InspectionGauge = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "mto_inspection_cache_size",
				Help: "Current size of the MTO inspection cache",
			},
		)
		TimeToInspectImageGivenHit = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "mto_cache_hit_processing_duration_seconds",
				Help:    "Duration to process a cache hit for MTO inspection",
				Buckets: utils.Buckets(),
			})
		TimeToInspectImageGivenMiss = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "mto_cache_miss_processing_duration_seconds",
				Help:    "Duration to process a cache miss for MTO inspection",
				Buckets: utils.Buckets(),
			})

		metrics2.Registry.MustRegister(InspectionGauge)
	})
}
