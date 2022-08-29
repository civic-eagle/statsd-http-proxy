package router

import (
	"fmt"
	"net/http"

	"github.com/civic-eagle/statsd-http-proxy/proxy/middleware"
	"github.com/julienschmidt/httprouter"
	log "github.com/sirupsen/logrus"
	vmmetrics "github.com/VictoriaMetrics/metrics"
)

// NewHTTPRouter creates julienschmidt's HTTP router
func NewHTTPRouter(
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
					func(w http.ResponseWriter, r *http.Request) {
						fmt.Fprint(w, "OK")
					},
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
		"/batch",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							// get variables from path
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalBatch(w, r, body)
						},
					),
					tokenSecret,
				),
			),
		),
	)

	/*
	There's a lot of "duplicate" code here, but it follows
	from a bug (https://github.com/julienschmidt/httprouter/issues/183)
	in the httprouter package from 2017.

	Until we can say:
	router.Handler(
		http.MethodPost,
		"/batch",
		...
	)
	router.Handler(
		http.MethodPost,
		"/:type",
		...
	)

	We'll need all these extra paths...
	*/
	router.Handler(
		http.MethodPost,
		"/count",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalMetric(w, r, body, "count")
						},
					),
					tokenSecret,
				),
			),
		),
	)

	router.Handler(
		http.MethodPost,
		"/gauge",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalMetric(w, r, body, "gauge")
						},
					),
					tokenSecret,
				),
			),
		),
	)

	router.Handler(
		http.MethodPost,
		"/timing",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalMetric(w, r, body, "timing")
						},
					),
					tokenSecret,
				),
			),
		),
	)

	router.Handler(
		http.MethodPost,
		"/set",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							// get variables from path
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalMetric(w, r, body, "set")
						},
					),
					tokenSecret,
				),
			),
		),
	)

	router.Handler(
		http.MethodPost,
		"/count/:metric",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							// get variables from path
							params := httprouter.ParamsFromContext(r.Context())
							metricName := params.ByName("metric")
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalMetricName(w, r, body, "count", metricName)
						},
					),
					tokenSecret,
				),
			),
		),
	)

	router.Handler(
		http.MethodPost,
		"/gauge/:metric",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							// get variables from path
							params := httprouter.ParamsFromContext(r.Context())
							metricName := params.ByName("metric")
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalMetricName(w, r, body, "gauge", metricName)
						},
					),
					tokenSecret,
				),
			),
		),
	)

	router.Handler(
		http.MethodPost,
		"/timing/:metric",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							// get variables from path
							params := httprouter.ParamsFromContext(r.Context())
							metricName := params.ByName("metric")
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalMetricName(w, r, body, "timing", metricName)
						},
					),
					tokenSecret,
				),
			),
		),
	)

	router.Handler(
		http.MethodPost,
		"/set/:metric",
		middleware.Instrument(
			middleware.ValidateCORS(
				middleware.ValidateJWT(
					http.HandlerFunc(
						func(w http.ResponseWriter, r *http.Request) {
							// get variables from path
							params := httprouter.ParamsFromContext(r.Context())
							metricName := params.ByName("metric")
							body, err := procBody(r)
							if err != nil {
								http.Error(w, err.Error(), 400)
								return
							}
							unMarshalMetricName(w, r, body, "set", metricName)
						},
					),
					tokenSecret,
				),
			),
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
