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
	h := Instrument(nextHandler)
	request := httptest.NewRequest("GET", "http://testing/healthcheck", nil)
	responseWriter := httptest.NewRecorder()
	h.ServeHTTP(responseWriter, request)
	response := responseWriter.Result()
	rt := require.New(t)

	rt.Equal(http.StatusOK, response.StatusCode)
	metrics := vmmetrics.ListMetricNames()
	rt.Equal(2, len(metrics))
}
