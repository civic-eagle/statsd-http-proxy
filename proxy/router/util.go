package router

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/civic-eagle/statsd-http-proxy/proxy/config"
)


func unMarshalBatch(w http.ResponseWriter, r *http.Request, body []byte) {
	var reqs config.BatchRequest
	if err := json.Unmarshal(body, &reqs); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	for _, metric := range reqs.Metrics {
		metric.MetricType = reqs.MetricType
		config.ProcessChan <- metric
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
