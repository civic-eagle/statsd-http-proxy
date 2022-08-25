package routehandler

import (
	"fmt"
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

// multiMetric: Allow for writing multiple stats at once in a batch
type multiMetric struct {
	Metrics []MetricRequest
}

// MetricRequest: internal representation of a metric to be written
type MetricRequest struct {
	Metric	string `json:"metric,omitempty"`
	Value    interface{} `json:"value"`
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

// HandleHeartbeatRequest: Just respond to health check requests
func (routeHandler *RouteHandler) HandleHeartbeatRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK")
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

	/*
	POSTs to '/:type' may have two different objects:
	Either an array of metrics to write (batch sending improves performance significantly!)
	or a single metric at a time.
	*/
	var reqs multiMetric
	if err := json.Unmarshal(body, &reqs); err == nil {
		for _, req := range reqs.Metrics {
			req, err = processMetric(req, routeHandler.metricPrefix, routeHandler.normalize, routeHandler.promFilter)
			if err != nil {
				http.Error(w, err.Error(), 400)
				return
			}
			sendMetric(routeHandler, metricType, req.Metric, req.Value, float32(req.SampleRate))
		}
	} else {
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

func sendMetric(routeHandler *RouteHandler, metricType string, key string, value interface{}, sampleRate float32) {
	/*
	Since we have two incoming handler paths for metrics
	we need a common switch case to actually process each metric
	once we've formatted it consistently
	Simply actually increment the correct values in our internal
	statsd client (and bump related internal metrics)
	*/
	switch metricType {
	case "count":
		routeHandler.statsdClient.Count(key, value.(int), sampleRate)
		vmmetrics.GetOrCreateCounter("counters_added_total").Inc()
	case "gauge":
		routeHandler.statsdClient.Gauge(key, value.(int))
		vmmetrics.GetOrCreateCounter("gauges_added_total").Inc()
	case "timing":
		routeHandler.statsdClient.Timing(key, value.(int64), sampleRate)
		vmmetrics.GetOrCreateCounter("timing_added_total").Inc()
	case "set":
		routeHandler.statsdClient.Set(key, value.(int))
		vmmetrics.GetOrCreateCounter("set_added_total").Inc()
	}
}
