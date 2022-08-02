package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/negroni"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

// Handler returns an measuring standard http.Handler.
func Instrument(h http.Handler, proxyPrefix string) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			// get the path for labelling
			lrw := negroni.NewResponseWriter(w)
			var path string
			if proxyPrefix != "" && proxyPrefix != "/" {
				h = http.StripPrefix(proxyPrefix, h)
				path = strings.TrimPrefix(r.URL.Path, proxyPrefix)
			} else {
				path = r.URL.Path
			}
			method := r.Method

			// Start the timer and when finishing measure the duration.
			start := time.Now()
			defer func() {
				code := strconv.Itoa(lrw.Status())
				reqStr := fmt.Sprintf("http_requests_total{method=%q,path=%q,status_code=%q}", method, path, code)
				vmmetrics.GetOrCreateCounter(reqStr).Inc()
				takenStr := fmt.Sprintf("http_request_time_secs_total{method=%q,path=%q,status_code=%q}", method, path, code)
				vmmetrics.GetOrCreateSummary(takenStr).UpdateDuration(start)
			}()
			h.ServeHTTP(lrw, r)
		},
	)
}
