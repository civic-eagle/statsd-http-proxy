package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/urfave/negroni"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

// Handler returns an measuring standard http.Handler.
func Instrument(h http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// get the path for labelling
			lrw := negroni.NewResponseWriter(w)
			path := r.URL.Path
			method := r.Method

			// Start the timer and when finishing measure the duration.
			start := time.Now()
			defer func() {
				code := lrw.Status()
				reqStr := fmt.Sprintf("http_requests_total{method=%q,path=%q,status_code=%q}", method, path, code)
				//reqStr := fmt.Sprintf("http_requests_total{method=%q,path=%q}", method, path)
				vmmetrics.GetOrCreateCounter(reqStr).Inc()
				takenStr := fmt.Sprintf("http_request_time_secs_total{method=%q,path=%q,status_code=%q}", method, path, code)
				//takenStr := fmt.Sprintf("http_request_time_secs_total{method=%q,path=%q}", method, path)
				vmmetrics.GetOrCreateSummary(takenStr).UpdateDuration(start)
			}()
			h.ServeHTTP(lrw, r)
		},
	)
}
