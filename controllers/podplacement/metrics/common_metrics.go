package metrics

import (
	"sync"

	metrics2 "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

func buckets() []float64 {
	return []float64{
		// Values are in seconds
		0.001, 0.002, 0.005, 0.010, 0.020, 0.050, // exponential-like buckets for values < 0.1
		0.100, 0.200, 0.300, 0.400, 0.500, 0.600, // linear buckets for the 0.1 <= values < 1
		0.700, 0.800, 0.900, 1.000, 2.000, 4.000, // exponential buckets for values >= 1
	}
}

var GatedPodsGauge prometheus.Gauge
var onceCommon sync.Once

func initCommonMetrics() {
	onceCommon.Do(func() {
		GatedPodsGauge = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "mto_ppo_pods_gated",
				Help: "The current number of gated pods (this metric is not considered reliable yet)",
			},
		)
		metrics2.Registry.MustRegister(GatedPodsGauge)
	})
}
