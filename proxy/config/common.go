package config

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
)

func init() {
	ProcessChan = make(chan MetricRequest, 1000)
}
