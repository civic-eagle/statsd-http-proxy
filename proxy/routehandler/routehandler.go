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

func procBody(r *http.Request) ([]byte, error) {
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		return []byte{}, fmt.Errorf("Unsupported content type %v", r.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		return []byte{}, err
	}
	r.Body.Close()

	return body, nil
}

func processMetric(m MetricRequest, prefix string, normalize bool, promFilter bool) (MetricRequest, error) {
	var err error
	if prefix != "" {
		m.Metric = prefix + m.Metric
	}

	if normalize {
		m.Metric = strings.ToLower(m.Metric)
		m.Tags = strings.ToLower(m.Tags)
	}

	if promFilter {
		m, err = filterPromMetric(m)
		if err != nil {
			return MetricRequest{}, err
		}
	}
	if m.Tags != "" {
		m.Metric += processTags(m.Tags)
	}

	return m, nil
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

	var finalTags string
	for _, pair := range list {
		pairItems := strings.Split(pair, "=")
		if len(pairItems) != 2 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Missing pair")
			continue
		} else if len(strings.TrimSpace(pairItems[0])) == 0 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Invalid tag key")
			continue
		} else if len(strings.TrimSpace(pairItems[1])) == 0 {
			vmmetrics.GetOrCreateCounter("metrics_tags_dropped_total").Inc()
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Invalid tag value")
			continue
		}
		finalTags += fmt.Sprintf("%s,", pair)
	}
	return "," + strings.TrimSuffix(finalTags, ",")
}

func filterPromMetric(m MetricRequest) (MetricRequest, error) {
	/*
	Remove/Replace any characters that don't meet the Prometheus
	data model requirements: https://prometheus.io/docs/concepts/data_model/
	*/
	metric := MetricRequest{
		Metric: "", Value: m.Value, Tags: "", SampleRate: 0,
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
			var finalTags string
			for _, pair := range list {
				tagPair := strings.Split(pair, "=")
				// filter out any bad tag pairs first
				if len(tagPair) != 2 || len(strings.TrimSpace(tagPair[0])) == 0 || len(strings.TrimSpace(tagPair[1])) == 0 {
					log.WithFields(log.Fields{"Tags": list, "pair": tagPair}).Debug("Invalid tag set")
					continue
				}
				if !allowedTagKeys.MatchString(tagPair[0]) {
					tagKey := replaceChars.ReplaceAllString(tagPair[0], "_")
					finalTags += fmt.Sprintf("%s=%s,", tagKey, tagPair[1])
				} else {
					finalTags += fmt.Sprintf("%s,", pair)
				}
			}
			metric.Tags = strings.TrimSuffix(finalTags, ",")
		}
	} else {
		metric.Tags = ""
	}
	metric.SampleRate = m.SampleRate
	log.WithFields(log.Fields{"metric": metric}).Debug("Final Metric")
	return metric, nil
}
