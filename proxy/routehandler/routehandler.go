package routehandler

import (
	"fmt"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/civic-eagle/statsd-http-proxy/proxy/statsdclient"
	log "github.com/sirupsen/logrus"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

var (
	allowedNames     = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_:]*$")
	allowedFirstChar = regexp.MustCompile("^[a-zA-Z]")
	replaceChars     = regexp.MustCompile("[^a-zA-Z0-9_:]")
	allowedTagKeys   = regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_]*$")
)

// 5 MB
const maxBodySize = 5000 * 1024 * 1024

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
	Value    int    `json:"value"`
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

func (routeHandler *RouteHandler) HandleMetric(
	w http.ResponseWriter,
	r *http.Request,
	metricType string,
) {
	req, err := procBody(w, r, routeHandler.metricPrefix, routeHandler.promFilter, routeHandler.normalize)
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
	req, err := procBody(w, r, routeHandler.metricPrefix, routeHandler.promFilter, routeHandler.normalize)
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

func sendMetric(routeHandler *RouteHandler, metricType string, key string, value int, sampleRate float32) {
	/*
	Since we have two incoming handler paths for metrics
	we need a common switch case to actually process each metric
	once we've formatted it consistently
	Simply actually increment the correct values in our internal
	statsd client (and bump related internal metrics)
	*/
	switch metricType {
	case "count":
		routeHandler.statsdClient.Count(key, value, sampleRate)
		vmmetrics.GetOrCreateCounter("counters_added_total").Inc()
	case "gauge":
		routeHandler.statsdClient.Gauge(key, value)
		vmmetrics.GetOrCreateCounter("gauges_added_total").Inc()
	case "timing":
		routeHandler.statsdClient.Timing(key, int64(value), sampleRate)
		vmmetrics.GetOrCreateCounter("timing_added_total").Inc()
	case "set":
		routeHandler.statsdClient.Set(key, value)
		vmmetrics.GetOrCreateCounter("set_added_total").Inc()
	}
}

func procBody(w http.ResponseWriter, r *http.Request, prefix string, promFilter bool, normalize bool) (MetricRequest, error) {
	/*
	All incoming POST requests to '/:type'
	and '/:type/:metric' should have consistent
	data (JSON object) that we can parse
	*/
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(w, fmt.Sprintf("Unsupported content type %v", r.Header.Get("Content-Type")), 400)
		return MetricRequest{}, fmt.Errorf("Unsupported content type")
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return MetricRequest{}, err
	}
	r.Body.Close()

	var req MetricRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return MetricRequest{}, err
	}

	if prefix != "" {
		req.Metric = prefix + req.Metric
	}

	if normalize {
		req.Metric = strings.ToLower(req.Metric)
		req.Tags = strings.ToLower(req.Tags)
	}

	if promFilter {
		return filterMetric(req)
	} else {
		return req, nil
	}
}

func processTags(tagsList string) string {
	/*
	Process tags for a metric.
	Starting with a string of comma-separated key=value pairs,
	we need to ensure that:
	1. each pair contains two items, one on each side of a '='
	2. that both the key and the value strings have actual content ('=value' and 'key=' are not allowed)

	Then return the resulting pairs to a string

	Any failures means the metric gets no tags
	*/
	list := strings.Split(strings.TrimSpace(tagsList), ",")
	if len(list) == 0 {
		return ""
	}

	for _, pair := range list {
		pairItems := strings.Split(pair, "=")
		if len(pairItems) != 2 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Missing pair")
			return ""
		} else if len(strings.TrimSpace(pairItems[0])) == 0 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Invalid tag key")
			return ""
		} else if len(strings.TrimSpace(pairItems[1])) == 0 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Invalid tag value")
			return ""
		}
	}
	return "," + tagsList
}

func filterMetric(m MetricRequest) (MetricRequest, error) {
	metric := MetricRequest{
		Metric: "", Value: 0, Tags: "", SampleRate: 0,
	}
	if !allowedFirstChar.MatchString(m.Metric) {
		vmmetrics.GetOrCreateCounter("metrics_dropped_total").Inc()
		return MetricRequest{}, fmt.Errorf("Invalid first character in metric name")
	}
	if !allowedNames.MatchString(m.Metric) {
		metric.Metric = replaceChars.ReplaceAllString(m.Metric, "_")
	} else {
		metric.Metric = m.Metric
	}
	if m.Tags != "" {
		list := strings.Split(strings.TrimSpace(m.Tags), ",")
		if len(list) > 0 {
			finalTags := ""
			for _, pair := range list {
				tagPair := strings.Split(pair, "=")
				// filter out any bad tag pairs first
				if len(tagPair) != 2 || len(strings.TrimSpace(tagPair[0])) == 0 || len(strings.TrimSpace(tagPair[1])) == 0 {
					continue
				}
				if !allowedTagKeys.MatchString(tagPair[0]) {
					tagKey := replaceChars.ReplaceAllString(tagPair[0], "_")
					finalTags += fmt.Sprintf("%s=%s", tagKey, tagPair[1])
				} else {
					finalTags += pair
				}
			}
			metric.Tags = finalTags
		}
	} else {
		metric.Tags = ""
	}
	metric.Value = m.Value
	metric.SampleRate = m.SampleRate

	return metric, nil
}
