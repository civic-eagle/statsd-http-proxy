package router

import (
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

	// register http request handlers
	router.Handler(
		http.MethodGet,
		"/heartbeat",
		middleware.ProxyCleanup(
			middleware.Instrument(
				middleware.ValidateCORS(
					http.HandlerFunc(
						routeHandler.HandleHeartbeatRequest,
					),
				),
			),
			proxyPath,
		),
	)

	router.Handler(
		http.MethodGet,
		"/metrics",
		middleware.ProxyCleanup(
			middleware.Instrument(
				middleware.ValidateCORS(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							vmmetrics.WritePrometheus(w, true)
						},
					),
				),
			),
			proxyPath,
		),
	)

	router.Handler(
		http.MethodPost,
		"/:type/",
		middleware.ProxyCleanup(
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
			),
			proxyPath,
		),
	)

	// Handle pre-flight CORS requests
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{"path": r.URL.Path}).Info("Pre-flight function")
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
