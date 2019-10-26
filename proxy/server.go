package proxy

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GoMetric/go-statsd-client"
	"github.com/GoMetric/statsd-http-proxy/proxy/routehandler"
	"github.com/GoMetric/statsd-http-proxy/proxy/router"
)

// Server is a proxy server between HTTP REST API and UDP Connection to StatsD
type Server struct {
	httpAddress  string
	httpServer   *http.Server
	statsdClient *statsd.Client
	tlsCert      string
	tlsKey       string
}

// NewServer creates new instance of StatsD HTTP Proxy
func NewServer(
	httpHost string,
	httpPort int,
	httpReadTimeout int,
	httpWriteTimeout int,
	httpIdleTimeout int,
	statsdHost string,
	statsdPort int,
	tlsCert string,
	tlsKey string,
	metricPrefix string,
	tokenSecret string,
	verbose bool,
) *Server {
	// configure logging
	var logOutput io.Writer
	if verbose == true {
		logOutput = os.Stderr
	} else {
		logOutput = ioutil.Discard
	}

	log.SetOutput(logOutput)

	logger := log.New(logOutput, "", log.LstdFlags)

	// create StatsD Client
	statsdClient := statsd.NewClient(statsdHost, statsdPort)

	// build route handler
	routeHandler := routehandler.NewRouteHandler(
		statsdClient,
		metricPrefix,
	)

	// build router
	router := router.NewGorillaMuxRouter(routeHandler, tokenSecret)

	// get HTTP server address to bind
	httpAddress := fmt.Sprintf("%s:%d", httpHost, httpPort)

	// create http server
	httpServer := &http.Server{
		Addr:           httpAddress,
		Handler:        router,
		ErrorLog:       logger,
		ReadTimeout:    time.Duration(httpReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(httpWriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(httpIdleTimeout) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	statsdHTTPProxyServer := Server{
		httpAddress,
		httpServer,
		statsdClient,
		tlsCert,
		tlsKey,
	}

	return &statsdHTTPProxyServer
}

// Listen starts listening HTTP connections
func (proxyServer *Server) Listen() {
	// prepare for gracefull shutdown
	gracefullStopSignalHandler := make(chan os.Signal, 1)
	signal.Notify(gracefullStopSignalHandler, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// start HTTP/HTTPS proxy to StatsD
	go func() {
		log.Printf("Starting HTTP server at %s", proxyServer.httpAddress)

		// open StatsD connection
		proxyServer.statsdClient.Open()
		defer proxyServer.statsdClient.Close()

		// open HTTP connection
		var err error
		if len(proxyServer.tlsCert) > 0 && len(proxyServer.tlsKey) > 0 {
			err = proxyServer.httpServer.ListenAndServeTLS(proxyServer.tlsCert, proxyServer.tlsKey)
		} else {
			err = proxyServer.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-gracefullStopSignalHandler

	// Gracefull shutdown
	log.Printf("Stopping HTTP server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := proxyServer.httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP Server Shutdown Failed:%+v", err)
	}

	log.Printf("HTTP server stopped successfully")
}