package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
				log.WithFields(log.Fields{"prefix": proxyPrefix, "url": r.URL.Path}).Debug("Path To process")
				// the prefix is already formatted in the calling function, so we can just leverage it here
				path = strings.TrimPrefix(r.URL.Path, fmt.Sprintf("/%s", proxyPrefix))
				log.WithFields(log.Fields{"result": path}).Debug("Final path to write in metric name")
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
