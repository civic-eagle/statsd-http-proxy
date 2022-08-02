package router

import (
	"fmt"
	"net/http"

	"github.com/civic-eagle/statsd-http-proxy/proxy/middleware"
	"github.com/civic-eagle/statsd-http-proxy/proxy/routehandler"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

// NewHTTPRouter creates julienschmidt's HTTP router
func NewHTTPRouter(
	routeHandler *routehandler.RouteHandler,
	proxyPath string,
	tokenSecret string,
) http.Handler {
	// build router
	router := httprouter.New()
	var rootPath string
	if proxyPath != "" && proxyPath != "/" {
		rootPath = fmt.Sprintf("/%s", proxyPath)
	} else {
		rootPath = ""
	}
	log.WithFields(log.Fields{"heartbeat": fmt.Sprintf("%s/heartbeat", rootPath)}).Debug("heartbeat route")
	log.WithFields(log.Fields{"metrics": fmt.Sprintf("%s/metrics", rootPath)}).Debug("metrics route")
	log.WithFields(log.Fields{"stats": fmt.Sprintf("%s/:type", rootPath)}).Debug("actual stats routes")

	// register http request handlers
	router.Handler(
		http.MethodGet,
		fmt.Sprintf("%s/heartbeat", rootPath),
		middleware.Instrument(
			middleware.ValidateCORS(
				http.HandlerFunc(
					routeHandler.HandleHeartbeatRequest,
				),
			),
			rootPath,
		),
	)

	router.Handler(
		http.MethodGet,
		fmt.Sprintf("%s/metrics", rootPath),
		middleware.Instrument(
			middleware.ValidateCORS(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						vmmetrics.WritePrometheus(w, true)
					},
				),
			),
			rootPath,
		),
	)

	router.Handler(
		http.MethodPost,
		fmt.Sprintf("%s/:type", rootPath),
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							// get variables from path
							params := httprouter.ParamsFromContext(r.Context())
							metricType := params.ByName("type")

							routeHandler.HandleMetric(w, r, metricType)
						},
					),
					tokenSecret,
				),
			),
			rootPath,
		),
	)

	// Handle pre-flight CORS requests
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{"path": r.URL.Path}).Debug("Pre-flight function")
		if r.Header.Get("Access-Control-Request-Method") == "" {
			return
		}

		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Add("Access-Control-Allow-Origin", origin)
			w.Header().Add("Access-Control-Allow-Headers", middleware.JwtHeaderName+", X-Requested-With, Origin, Accept, Content-Type, Authentication")
			w.Header().Add("Access-Control-Allow-Methods", "GET, POST, HEAD, OPTIONS")
		}

		// Adjust status code to 204
		w.WriteHeader(http.StatusNoContent)
	})

	return router
}
