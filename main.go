package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strings"
	"time"

	vmmetrics "github.com/VictoriaMetrics/metrics"
	"github.com/civic-eagle/statsd-http-proxy/proxy"
	"github.com/civic-eagle/statsd-http-proxy/proxy/process"
	"github.com/civic-eagle/statsd-http-proxy/proxy/statsdclient"
	log "github.com/sirupsen/logrus"
)

// Version is a current git tag
// Injected by compilation flag
var Version = "Unknown"

// BuildDate is a date of build
// Injected by compilation flag
var BuildTime = "Unknown"

// BuildUser is the user that built
// Injected by compilation flag
var BuildUser = "Unknown"

// HTTP connection params
const defaultHTTPHost = "127.0.0.1"
const defaultHTTPPort = 8825
const defaultHTTPReadTimeout = 2
const defaultHTTPWriteTimeout = 2
const defaultHTTPIdleTimeout = 5

// StatsD connection params
const defaultStatsDHost = "127.0.0.1"
const defaultStatsDPort = 8125

func main() {
	// metric instantiation (for global metrics we always want to see)
	startTime := time.Now()
	_ = vmmetrics.NewGauge("app_uptime_secs_total",
		func() float64 {
			return float64(time.Since(startTime).Seconds())
		})

	// logging configuration
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)

	// declare command line options
	var httpHost = flag.String("http-host", defaultHTTPHost, "HTTP listening address")
	var httpPort = flag.Int("http-port", defaultHTTPPort, "HTTP Port")
	var httpReadTimeout = flag.Int("http-timeout-read", defaultHTTPReadTimeout, "The maximum duration in seconds for reading the entire request, including the body")
	var httpWriteTimeout = flag.Int("http-timeout-write", defaultHTTPWriteTimeout, "The maximum duration in seconds before timing out writes of the response")
	var httpIdleTimeout = flag.Int("http-timeout-idle", defaultHTTPIdleTimeout, "The maximum amount of time in seconds to wait for the next request when keep-alives are enabled")
	var tlsCert = flag.String("tls-cert", "", "TLS certificate to enable HTTPS")
	var tlsKey = flag.String("tls-key", "", "TLS private key  to enable HTTPS")
	var statsdHost = flag.String("statsd-host", defaultStatsDHost, "StatsD listening address")
	var statsdPort = flag.Int("statsd-port", defaultStatsDPort, "StatsD Port")
	var metricPrefix = flag.String("metric-prefix", "", "Prefix of metric name")
	var tokenSecret = flag.String("jwt-secret", "", "Secret to encrypt JWT")
	var verbose = flag.Bool("verbose", false, "Verbose")
	var promFilter = flag.Bool("prometheus-compat", false, "Enforce prometheus data model compatibility on incoming metrics")
	var normalize = flag.Bool("normalize", false, "Ensure all metrics (and tags) are lower case strings")
	var version = flag.Bool("version", false, "Show version")
	var profilerHTTPort = flag.Int("profiler-http-port", 0, "Start profiler localhost")

	// get flags
	flag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}

	// show version and exit
	if *version {
		log.WithFields(log.Fields{"Version": Version, "BuildTime": BuildTime, "BuildUser": BuildUser}).Info("Version Info")
		os.Exit(0)
	}

	// start profiler
	if *profilerHTTPort > 0 {
		// enable block profiling
		runtime.SetBlockProfileRate(1)

		// start debug server
		profilerHTTPAddress := fmt.Sprintf("localhost:%d", *profilerHTTPort)
		go func() {
			log.WithFields(log.Fields{"Address": profilerHTTPAddress}).Info("Profiler started")
			log.Info(fmt.Sprintf("Open 'http://%s/debug/pprof/' in you browser or use 'go tool pprof http://%s/debug/pprof/heap' from console", profilerHTTPAddress, profilerHTTPAddress))
			log.Info("See details about pprof in https://golang.org/pkg/net/http/pprof/")
			log.Info(http.ListenAndServe(profilerHTTPAddress, nil))
		}()
	}

	/*
		Create processor and related tooling
		Because this is async, we create it outside the
		http server
	*/
	// prepare metric prefix
	if *metricPrefix != "" && (*metricPrefix)[len(*metricPrefix)-1:] != "_" {
		*metricPrefix = *metricPrefix + "_"
	}
	if *normalize {
		*metricPrefix = strings.ToLower(*metricPrefix)
	}

	// create StatsD Client
	statsdClient := statsdclient.NewGoMetricClient(*statsdHost, *statsdPort)
	// open StatsD connection
	statsdClient.Open()
	defer statsdClient.Close()

	// build processor
	processor := processor.NewProcessor(
		statsdClient,
		*metricPrefix,
		*promFilter,
		*normalize,
	)

	/*
		Multiple processing threads so we don't get bottle-necked on one processor
		Since each individual object on the channel is unique, we don't need
		state between processor threads!
	*/
	for thread := 1; thread <= 4; thread++ {
		go processor.Process()
	}

	// start proxy server
	proxyServer := proxy.NewServer(
		*httpHost,
		*httpPort,
		*httpReadTimeout,
		*httpWriteTimeout,
		*httpIdleTimeout,
		*tlsCert,
		*tlsKey,
		*tokenSecret,
		*verbose,
	)

	proxyServer.Listen()
}
