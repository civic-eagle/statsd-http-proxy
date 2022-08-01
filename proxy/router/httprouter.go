package router

import (
	"net/http"
	"strings"

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
		middleware.Instrument(
			middleware.ValidateCORS(
				http.HandlerFunc(
					routeHandler.HandleHeartbeatRequest,
				),
			),
		),
	)

	router.Handler(
		http.MethodGet,
		"/metrics",
		middleware.Instrument(
			middleware.ValidateCORS(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						vmmetrics.WritePrometheus(w, true)
					},
				),
			),
		),
	)

	router.Handler(
		http.MethodPost,
		"/:type/",
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
	)

	// Handle pre-flight CORS requests
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.WithFields(log.Fields{"path": r.URL.Path}).Info("Pre-flight function")
		if r.Header.Get("Access-Control-Request-Method") == "" {
			return
		}

		// if we have a proxy that doesn't remove proxy paths, define the path to remove
		if proxyPath != "" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, proxyPath)
			log.WithFields(log.Fields{"Prefix": proxyPath, "URL": r.URL.Path}).Debug("Trimmed proxy path")
		}
		// pathMetric := fmt.Sprintf(`http_requests_total{path=%q,
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
