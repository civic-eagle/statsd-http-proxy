package config

import (
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

// MetricRequest: internal representation of a metric to be written
type MetricRequest struct {
	Metric	string `json:"metric,omitempty"`
	Value    int64 `json:"value"`
	Tags string `json:"tags"`
	MetricType string `json:"metric_type,omitempty"`
	SampleRate float64 `json:"sampleRate"`
}

// 5 MB
const MaxBodySize = 5000 * 1024 * 1024

var (
	ProcessChan chan MetricRequest
	DroppedMetrics = vmmetrics.NewCounter("metrics_dropped_total")
)

func init() {
	ProcessChan = make(chan MetricRequest, 1000)

	// gauge instantiation (for global gauges we always want to see)
	_ = vmmetrics.NewGauge("processing_queue_length",
		func() float64 {
			return float64(len(ProcessChan))
		})
}
