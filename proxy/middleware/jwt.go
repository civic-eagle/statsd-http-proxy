package middleware

import (
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt"
	log "github.com/sirupsen/logrus"
)

const jwtQueryStringKeyName = "token"

const JwtHeaderName = "X-JWT-Token"

// validate JWT middleware
func ValidateJWT(next http.Handler, tokenSecret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if tokenSecret == "" {
			next.ServeHTTP(w, r)
		} else {
			// get JWT from header
			tokenString := r.Header.Get(JwtHeaderName)

			// get JWT from query string
			if tokenString == "" {
				tokenString = r.URL.Query().Get(jwtQueryStringKeyName)
			}

			if tokenString == "" {
				log.Error("Token not specified")
				http.Error(w, "Token not specified", 401)
				return
			}

			// parse JWT
			_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					log.Error("Bad signing format")
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(tokenSecret), nil
			})

			if err != nil {
				log.Error("Error parsing token")
				http.Error(w, "Error parsing token", 403)
				return
			}

			// accept request
			next.ServeHTTP(w, r)
		}
	})
}
