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
	metricKey string,
) {
	switch metricType {
	case "count":
		routeHandler.handleCountRequest(w, r, metricKey)
	case "gauge":
		routeHandler.handleGaugeRequest(w, r, metricKey)
	case "timing":
		routeHandler.handleTimingRequest(w, r, metricKey)
	case "set":
		routeHandler.handleSetRequest(w, r, metricKey)
	}
}

func (routeHandler *RouteHandler) HandleHeartbeatRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "OK")
}
