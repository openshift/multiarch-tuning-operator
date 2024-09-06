package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func HistogramObserve(initialTime time.Time, histogram prometheus.Histogram) {
	histogram.Observe(time.Since(initialTime).Seconds())
}
