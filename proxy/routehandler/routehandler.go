package routehandler

import (
	"fmt"
	"net/http"

	"github.com/civic-eagle/statsd-http-proxy/proxy/statsdclient"
	vmmetrics "github.com/VictoriaMetrics/metrics"
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
	switch metricType {
	case "count":
		routeHandler.handleCountRequest(w, r)
	case "gauge":
		routeHandler.handleGaugeRequest(w, r)
	case "timing":
		routeHandler.handleTimingRequest(w, r)
	case "set":
		routeHandler.handleSetRequest(w, r)
	}
}

func (routeHandler *RouteHandler) HandleHeartbeatRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK")
	vmmetrics.GetOrCreateCounter(`http_requests_total{endpoint="/heartbeat",method="GET",status_code="200"}`).Inc()
}
