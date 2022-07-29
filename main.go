package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	"github.com/civic-eagle/statsd-http-proxy/proxy"
	"github.com/civic-eagle/statsd-http-proxy/proxy/stats"
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
const defaultHTTPReadTimeout = 1
const defaultHTTPWriteTimeout = 1
const defaultHTTPIdleTimeout = 1
const defaultStatsHTTPHost = "127.0.0.1"
const defaultStatsHTTPPort = "9991"

// StatsD connection params
const defaultStatsDHost = "127.0.0.1"
const defaultStatsDPort = 8125

func main() {
	// declare command line options
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	var httpHost = flag.String("http-host", defaultHTTPHost, "HTTP Host")
	var httpPort = flag.Int("http-port", defaultHTTPPort, "HTTP Port")
	var httpReadTimeout = flag.Int("http-timeout-read", defaultHTTPReadTimeout, "The maximum duration in seconds for reading the entire request, including the body")
	var httpWriteTimeout = flag.Int("http-timeout-write", defaultHTTPWriteTimeout, "The maximum duration in seconds before timing out writes of the response")
	var httpIdleTimeout = flag.Int("http-timeout-idle", defaultHTTPIdleTimeout, "The maximum amount of time in seconds to wait for the next request when keep-alives are enabled")
	var tlsCert = flag.String("tls-cert", "", "TLS certificate to enable HTTPS")
	var tlsKey = flag.String("tls-key", "", "TLS private key  to enable HTTPS")
	var statsdHost = flag.String("statsd-host", defaultStatsDHost, "StatsD Host")
	var statsdPort = flag.Int("statsd-port", defaultStatsDPort, "StatsD Port")
	var metricPrefix = flag.String("metric-prefix", "", "Prefix of metric name")
	var tokenSecret = flag.String("jwt-secret", "", "Secret to encrypt JWT")
	var verbose = flag.Bool("verbose", false, "Verbose")
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
			log.WithFields(log.Fields{}).Info(fmt.Sprintf("Open 'http://" + profilerHTTPAddress + "/debug/pprof/' in you browser or use 'go tool pprof http://" + profilerHTTPAddress + "/debug/pprof/heap' from console"))
			log.Info("See details about pprof in https://golang.org/pkg/net/http/pprof/")
			log.Info(http.ListenAndServe(profilerHTTPAddress, nil))
		}()
	}
	go stats.StatsListener(defaultStatsHTTPHost, defaultStatsHTTPPort)

	// start proxy server
	proxyServer := proxy.NewServer(
		*httpHost,
		*httpPort,
		*httpReadTimeout,
		*httpWriteTimeout,
		*httpIdleTimeout,
		*statsdHost,
		*statsdPort,
		*tlsCert,
		*tlsKey,
		*metricPrefix,
		*tokenSecret,
		*verbose,
	)

	proxyServer.Listen()
}
