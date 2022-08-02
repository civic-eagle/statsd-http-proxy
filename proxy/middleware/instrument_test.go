package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

func TestValidateInstrumentationWithoutProxy(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := Instrument(nextHandler, "")
	request := httptest.NewRequest("GET", "http://testing/healthcheck", nil)
	responseWriter := httptest.NewRecorder()
	h.ServeHTTP(responseWriter, request)
	response := responseWriter.Result()
	rt := require.New(t)

	rt.Equal(http.StatusOK, response.StatusCode)
	metrics := vmmetrics.ListMetricNames()
	rt.Equal(7, len(metrics))
}

func TestValidateInstrumentationWithProxy(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := Instrument(nextHandler, "/statsd")
	request := httptest.NewRequest("GET", "http://testing/statsd/healthcheck", nil)
	responseWriter := httptest.NewRecorder()
	h.ServeHTTP(responseWriter, request)
	response := responseWriter.Result()
	rt := require.New(t)

	rt.Equal(http.StatusOK, response.StatusCode)
	metrics := vmmetrics.ListMetricNames()
	rt.Equal(7, len(metrics))
}

func TestValidateInstrumentationWithProxyBadPath(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	h := Instrument(nextHandler, "/statsd")
	request := httptest.NewRequest("GET", "http://testing/healthcheck", nil)
	responseWriter := httptest.NewRecorder()
	h.ServeHTTP(responseWriter, request)
	response := responseWriter.Result()
	rt := require.New(t)

	// root path shouldn't work now
	rt.Equal(http.StatusNotFound, response.StatusCode)
	metrics := vmmetrics.ListMetricNames()
	// should return new stats for a 404 code, so duplicate all stats
	rt.Equal(14, len(metrics))
}
