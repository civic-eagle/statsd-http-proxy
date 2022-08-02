package middleware

import (
	"net/http"
)

// validate CORS headers
func ValidateCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			// allow any origin
			w.Header().Add("Access-Control-Allow-Origin", origin)

			// handle pre-flight OPTIONS request
			if r.Method == http.MethodOptions {
				if r.Header.Get("Access-Control-Request-Method") != "" {
					w.Header().Add("Access-Control-Allow-Methods", "GET, POST, HEAD, OPTIONS")
				}

				if r.Header.Get("Access-Control-Request-Headers") != "" {
					w.Header().Add("Access-Control-Allow-Headers", JwtHeaderName+", X-Requested-With, Origin, Accept, Content-Type, Authentication")
				}

				w.WriteHeader(http.StatusNoContent)

				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
