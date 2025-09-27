package utils

import (
	"os"
	"path"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func NewPtr[T any](a T) *T {
	return &a
}

func ArchLabelValue(arch string) string {
	return path.Join(LabelGroup, arch)
}

func HistogramObserve(initialTime time.Time, histogram prometheus.Histogram) {
	histogram.Observe(time.Since(initialTime).Seconds())
}

func Buckets() []float64 {
	return []float64{
		// Values are in seconds
		0.001, 0.002, 0.005, 0.010, 0.020, 0.050, // exponential-like buckets for values < 0.1
		0.100, 0.200, 0.300, 0.400, 0.500, 0.600, // linear buckets for the 0.1 <= values < 1
		0.700, 0.800, 0.900, 1.000, 2.000, 4.000, // exponential buckets for values >= 1
	}
}

func ShouldStdErr(fn func() error) {
	if err := fn(); err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
	}
}
