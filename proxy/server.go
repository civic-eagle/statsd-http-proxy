package proxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/civic-eagle/statsd-http-proxy/proxy/router"
	"github.com/civic-eagle/statsd-http-proxy/proxy/config"
	log "github.com/sirupsen/logrus"
)

// Server is a proxy server between HTTP REST API and UDP Connection to StatsD
type Server struct {
	httpAddress string
	httpServer  *http.Server
	tlsCert     string
	tlsKey      string
}

// NewServer creates new instance of StatsD HTTP Proxy
func NewServer(
	httpHost string,
	httpPort int,
	httpReadTimeout int,
	httpWriteTimeout int,
	httpIdleTimeout int,
	tlsCert string,
	tlsKey string,
	tokenSecret string,
	verbose bool,
) *Server {
	// build router
	httpServerHandler := router.NewHTTPRouter(tokenSecret)

	// get HTTP server address to bind
	httpAddress := fmt.Sprintf("%s:%d", httpHost, httpPort)

	// create http server
	httpServer := &http.Server{
		Addr:           httpAddress,
		Handler:        httpServerHandler,
		ReadTimeout:    time.Duration(httpReadTimeout) * time.Second,
		WriteTimeout:   time.Duration(httpWriteTimeout) * time.Second,
		IdleTimeout:    time.Duration(httpIdleTimeout) * time.Second,
		MaxHeaderBytes: 1 << 11,
	}

	statsdHTTPProxyServer := Server{
		httpAddress,
		httpServer,
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
		log.WithFields(log.Fields{"Address": proxyServer.httpAddress}).Info("Starting HTTP server")

		// open HTTP connection
		var err error
		if len(proxyServer.tlsCert) > 0 && len(proxyServer.tlsKey) > 0 {
			err = proxyServer.httpServer.ListenAndServeTLS(proxyServer.tlsCert, proxyServer.tlsKey)
		} else {
			err = proxyServer.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.WithFields(log.Fields{"Error": err}).Fatal("Cannot start HTTP Server")
		}
	}()

	<-gracefullStopSignalHandler

	// Gracefull shutdown
	log.Info("Stopping HTTP server")
	close(config.ProcessChan)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := proxyServer.httpServer.Shutdown(ctx); err != nil {
		log.WithFields(log.Fields{"error": err}).Fatal("HTTP Server Shutdown Failed")
	}

	log.Info("HTTP server stopped successfully")
}
