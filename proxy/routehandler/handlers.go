package routehandler

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/civic-eagle/statsd-http-proxy/proxy/stats"
	log "github.com/sirupsen/logrus"
)

type CountRequest struct {
	Metric	string `json:"metric"`
	Value    int    `json:"value,omitempty"`
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

func (routeHandler *RouteHandler) handleCountRequest(w http.ResponseWriter, r *http.Request) {
	body, err := procBody(w, r)
	if err != nil {
		return
	}
	var req CountRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var key = req.Metric + processTags(req.Tags)

	var sampleRate float64 = 1
	if req.SampleRate != 0 {
		sampleRate = float64(req.SampleRate)
	}
	routeHandler.statsdClient.Count(key, req.Value, float32(sampleRate))
	stats.CountersAdded.Inc()
}

type GaugeRequest struct {
	Metric	string `json:"metric"`
	Value int    `json:"value,omitempty"`
	Tags  string `json:"tags,omitempty"`
}

func (routeHandler *RouteHandler) handleGaugeRequest(w http.ResponseWriter, r *http.Request) {
	body, err := procBody(w, r)
	if err != nil {
		return
	}

	var req GaugeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var key = req.Metric + processTags(req.Tags)

	routeHandler.statsdClient.Gauge(key, req.Value)
	stats.GaugesAdded.Inc()
}

type TimingRequest struct {
	Metric	string `json:"metric"`
	Value int64    `json:"value,omitempty"`
	Tags     string `json:"tags,omitempty"`
	SampleRate float64 `json:"sampleRate"`
}

func (routeHandler *RouteHandler) handleTimingRequest(w http.ResponseWriter, r *http.Request) {
	body, err := procBody(w, r)
	if err != nil {
		return
	}

	var req TimingRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var key = req.Metric + processTags(req.Tags)

	var sampleRate float64 = 1
	if req.SampleRate != 0 {
		sampleRate = float64(req.SampleRate)
	}

	routeHandler.statsdClient.Timing(key, req.Value, float32(sampleRate))
	stats.TimingAdded.Inc()
}

type SetRequest struct {
	Metric	string `json:"metric"`
	Value int `json:"value,omitempty"`
	Tags  string `json:"tags,omitempty"`
}

func (routeHandler *RouteHandler) handleSetRequest(w http.ResponseWriter, r *http.Request) {
	body, err := procBody(w, r)
	if err != nil {
		return
	}

	var req SetRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	var key = req.Metric + processTags(req.Tags)

	routeHandler.statsdClient.Set(key, req.Value)
	stats.SetAdded.Inc()
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
