package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/civic-eagle/statsd-http-proxy/proxy/config"
	log "github.com/sirupsen/logrus"
)

func unMarshalBatch(w http.ResponseWriter, r *http.Request, body []byte) {
	var reqs []config.MetricRequest
	if err := json.Unmarshal(body, &reqs); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	for _, m := range reqs {
		if m.MetricType == "" {
			log.WithFields(log.Fields{"metric": m}).Error("Metric in batch missing type, cannot forward")
			config.DroppedMetrics.Inc()
			continue
		}
		config.ProcessChan <- m
	}
}

func unMarshalMetric(w http.ResponseWriter, r *http.Request, body []byte, metricType string) {
	var req config.MetricRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	req.MetricType = metricType
	config.ProcessChan <- req
}

func unMarshalMetricName(w http.ResponseWriter, r *http.Request, body []byte, metricType string, metricName string) {
	var req config.MetricRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	req.Metric = metricName
	req.MetricType = metricType
	config.ProcessChan <- req
}

func procBody(r *http.Request) ([]byte, error) {
	if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
		return []byte{}, fmt.Errorf("Unsupported content type %v", r.Header.Get("Content-Type"))
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, config.MaxBodySize))
	if err != nil {
		return []byte{}, err
	}
	r.Body.Close()

	return body, nil
}
