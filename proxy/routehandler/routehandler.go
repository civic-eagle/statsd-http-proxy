package routehandler

import (
	"encoding/json"
	"net/http"

	"github.com/civic-eagle/statsd-http-proxy/proxy/statsdclient"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

// RouteHandler as a collection of route handlers
type RouteHandler struct {
	statsdClient statsdclient.StatsdClientInterface
	metricPrefix string
	promFilter bool
	normalize bool
}

// MetricRequest: internal representation of a metric to be written
type MetricRequest struct {
	Metric	string `json:"metric,omitempty"`
	Value    int64 `json:"value"`
	Tags string `json:"tags"`
	SampleRate float64 `json:"sampleRate"`
}

// NewRouteHandler creates collection of route handlers
func NewRouteHandler(
	statsdClient statsdclient.StatsdClientInterface,
	metricPrefix string,
	promFilter bool,
	normalize bool,
) *RouteHandler {
	// build route handler
	routeHandler := RouteHandler{
		statsdClient,
		metricPrefix,
		promFilter,
		normalize,
	}

	return &routeHandler
}

// HandleBatchMetric: New path addressing metrics send through /batch/:type
func (routeHandler *RouteHandler) HandleBatchMetric(
	w http.ResponseWriter,
	r *http.Request,
	metricType string,
) {
	body, err := procBody(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	var reqs []MetricRequest
	if err := json.Unmarshal(body, &reqs); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	for _, req := range reqs {
		req, err = processMetric(req, routeHandler.metricPrefix, routeHandler.normalize, routeHandler.promFilter)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		sendMetric(routeHandler, metricType, req.Metric, req.Value, float32(req.SampleRate))
	}
}

// HandleMetric: New path addressing metrics send through /:type
func (routeHandler *RouteHandler) HandleMetric(
	w http.ResponseWriter,
	r *http.Request,
	metricType string,
) {
	body, err := procBody(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	var req MetricRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	req, err = processMetric(req, routeHandler.metricPrefix, routeHandler.normalize, routeHandler.promFilter)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	sendMetric(routeHandler, metricType, req.Metric, req.Value, float32(req.SampleRate))
}

// HandleMetricName: Old path addressing metrics send through /:type/:name
func (routeHandler *RouteHandler) HandleMetricName(
	w http.ResponseWriter,
	r *http.Request,
	metricType string,
	metricName string,
) {
	body, err := procBody(r)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	
	var req MetricRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	/*
	The only change we have to make between the old pattern and the new pattern
	is that we need to set the Metric key *after* we load the data into memory

	Also, the old path won't support batch writes because we can only have one
	metric name per write.
	*/
	req.Metric = metricName
	req, err = processMetric(req, routeHandler.metricPrefix, routeHandler.normalize, routeHandler.promFilter)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	sendMetric(routeHandler, metricType, req.Metric, req.Value, float32(req.SampleRate))
}

func sendMetric(routeHandler *RouteHandler, metricType string, key string, value int64, sampleRate float32) {
	/*
	Since we have two incoming handler paths for metrics
	we need a common switch case to actually process each metric
	once we've formatted it consistently
	Simply actually increment the correct values in our internal
	statsd client (and bump related internal metrics)
	*/
	switch metricType {
	case "count":
		routeHandler.statsdClient.Count(key, int(value), sampleRate)
		vmmetrics.GetOrCreateCounter("counters_added_total").Inc()
	case "gauge":
		routeHandler.statsdClient.Gauge(key, int(value))
		vmmetrics.GetOrCreateCounter("gauges_added_total").Inc()
	case "timing":
		routeHandler.statsdClient.Timing(key, value, sampleRate)
		vmmetrics.GetOrCreateCounter("timing_added_total").Inc()
	case "set":
		routeHandler.statsdClient.Set(key, int(value))
		vmmetrics.GetOrCreateCounter("set_added_total").Inc()
	}
}
