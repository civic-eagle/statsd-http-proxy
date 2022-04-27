package router

import (
	"net/http"

	"github.com/civic-eagle/statsd-http-proxy/proxy/middleware"
	"github.com/civic-eagle/statsd-http-proxy/proxy/routehandler"
	"github.com/julienschmidt/httprouter"
)

// NewHTTPRouter creates julienschmidt's HTTP router
func NewHTTPRouter(
	routeHandler *routehandler.RouteHandler,
	tokenSecret string,
) http.Handler {
	// build router
	router := httprouter.New()

	// register http request handlers
	router.Handler(
		http.MethodGet,
		"/heartbeat",
		middleware.ValidateCORS(http.HandlerFunc(routeHandler.HandleHeartbeatRequest)))

	router.Handler(
		http.MethodPost,
		"/:type/:key",
		middleware.ValidateCORS(
			middleware.ValidateJWT(
				http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						// get variables from path
						params := httprouter.ParamsFromContext(r.Context())
						metricType := params.ByName("type")
						metricKeySuffix := params.ByName("key")

						routeHandler.HandleMetric(w, r, metricType, metricKeySuffix)
					},
				),
				tokenSecret,
			),
		),
	)

	// Handle pre-flight CORS requests
	router.GlobalOPTIONS = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
