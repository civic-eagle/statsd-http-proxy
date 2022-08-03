package routehandler

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

type MetricRequest struct {
	Metric	string `json:"metric,omitempty"`
	Value    int    `json:"value"`
	Tags string `json:"tags,omitempty"`
	SampleRate float64 `json:"sampleRate"`
}

// 5 MB
const maxBodySize = 5000 * 1024 * 1024

func procBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		http.Error(w, "Unsupported content type", 400)
		return []byte(""), fmt.Errorf("Unsupported content type")
	}

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, maxBodySize))
	if err != nil {
		http.Error(w, err.Error(), 400)
		return []byte(""), err
	}
	r.Body.Close()

	return body, nil
}

func processTags(tagsList string) string {
	list := strings.Split(strings.TrimSpace(tagsList), ",")
	if len(list) == 0 {
		return ""
	}

	for _, pair := range list {
		pairItems := strings.Split(pair, "=")
		if len(pairItems) != 2 {
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Missing pair")
			return ""
		} else if len(strings.TrimSpace(pairItems[0])) == 0 {
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Invalid tag key")
			return ""
		} else if len(strings.TrimSpace(pairItems[1])) == 0 {
			log.WithFields(log.Fields{"Tags": tagsList, "pair": pairItems}).Debug("Invalid tag value")
			return ""
		}
	}
	return "," + tagsList
}

func sendMetric(routeHandler *RouteHandler, metricType string, key string, value int, sampleRate float32) {
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
