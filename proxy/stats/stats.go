package stats

import (
	"fmt"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/metrics"
	log "github.com/sirupsen/logrus"
)

var (
	startTime = time.Now()
	CountersAdded = metrics.NewCounter("counters_added_total")
	GaugesAdded = metrics.NewCounter("gauges_added_total")
	TimingAdded = metrics.NewCounter("timing_added_total")
	SetAdded = metrics.NewCounter("set_added_total")

	JwtMissingToken = metrics.NewCounter("auth_reqs_without_token_total")
	JwtBadToken = metrics.NewCounter("auth_reqs_bad_token_total")

	//Uptime : since app start
	Uptime = metrics.NewGauge("app_uptime_secs_total",
		func() float64 {
			return float64(time.Since(startTime).Seconds())
		})
)

//StatsListener : Actually expose an endpoint for stats to be scraped
func StatsListener(address string, port string) {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics.WritePrometheus(w, true)
	})
	err := http.ListenAndServe(fmt.Sprintf("%v:%v", address, port), nil)
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("Couldn't start stats listener")
	}
}
