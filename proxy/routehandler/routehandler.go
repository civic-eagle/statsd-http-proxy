package routehandler

import (
	"fmt"
	"net/http"

	"github.com/civic-eagle/statsd-http-proxy/proxy/statsdclient"
)

// RouteHandler as a collection of route handlers
type RouteHandler struct {
	statsdClient statsdclient.StatsdClientInterface
	metricPrefix string
}

// NewRouteHandler creates collection of route handlers
func NewRouteHandler(
	statsdClient statsdclient.StatsdClientInterface,
	metricPrefix string,
) *RouteHandler {
	// build route handler
	routeHandler := RouteHandler{
		statsdClient,
		metricPrefix,
	}

	return &routeHandler
}


func (routeHandler *RouteHandler) HandleMetric(
	w http.ResponseWriter,
	r *http.Request,
	metricType string,
) {
	req, err := procBody(w, r)
	if err != nil {
		return
	}
	var key = req.Metric
	if req.Tags != "" {
		key += processTags(req.Tags)
	}

	var sampleRate float64 = 1
	if req.SampleRate != 0 {
		sampleRate = float64(req.SampleRate)
	}

	sendMetric(routeHandler, metricType, key, req.Value, float32(sampleRate))
}

func (routeHandler *RouteHandler) HandleMetricName(
	w http.ResponseWriter,
	r *http.Request,
	metricType string,
	metricName string,
) {
	req, err := procBody(w, r)
	if err != nil {
		return
	}
	var key = metricName
	if req.Tags != "" {
		key += processTags(req.Tags)
	}

	var sampleRate float64 = 1
	if req.SampleRate != 0 {
		sampleRate = float64(req.SampleRate)
	}

	sendMetric(routeHandler, metricType, key, req.Value, float32(sampleRate))
}

func (routeHandler *RouteHandler) HandleHeartbeatRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK")
}
