package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

const VALID_TOKEN_SECTET = "somesecret"
const VALID_TOKEN = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJzdGF0c2QtcmVzdC1zZXJ2ZXIiLCJpYXQiOjE1MDY5NzI1ODAsImV4cCI6MTg4NTY2Mzc4MCwiYXVkIjoiaHR0cHM6Ly9naXRodWIuY29tL3Nva2lsL3N0YXRzZC1yZXN0LXNlcnZlciIsInN1YiI6InNva2lsIn0.sOb0ccRBnN1u9IP2jhJrcNod14G5t-jMHNb_fsWov5c"

func TestValidateJWTWithNoTokenSecretInCliConfigured(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handlerWithJWTValidation := ValidateJWT(nextHandler, "")

	request := httptest.NewRequest("GET", "http://testing", nil)
	responseWriter := httptest.NewRecorder()

	handlerWithJWTValidation.ServeHTTP(responseWriter, request)
}

func TestValidateJWTWithNoTokenInHeaderAndQuery(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handlerWithJWTValidation := ValidateJWT(nextHandler, VALID_TOKEN_SECTET)

	request := httptest.NewRequest("GET", "http://testing", nil)
	responseWriter := httptest.NewRecorder()

	handlerWithJWTValidation.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	responseBody, _ := io.ReadAll(response.Body)

	require := require.New(t)

	require.Equal(401, response.StatusCode)
	require.Equal("Token not specified\n", string(responseBody))
}

func TestValidateJWTWithInvalidTokenInHeader(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handlerWithJWTValidation := ValidateJWT(nextHandler, VALID_TOKEN_SECTET)

	request := httptest.NewRequest("GET", "http://testing", nil)
	responseWriter := httptest.NewRecorder()

	request.Header.Add("X-JWT-Token", "some_invalid_token")

	handlerWithJWTValidation.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	responseBody, _ := io.ReadAll(response.Body)

	require := require.New(t)

	require.Equal(403, response.StatusCode)
	require.Equal("Error parsing token\n", string(responseBody))
}

func TestValidateJWTWithValidTokenInHeader(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handlerWithJWTValidation := ValidateJWT(nextHandler, VALID_TOKEN_SECTET)

	request := httptest.NewRequest("GET", "http://testing", nil)
	responseWriter := httptest.NewRecorder()

	request.Header.Add("X-JWT-Token", VALID_TOKEN)

	handlerWithJWTValidation.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	responseBody, _ := io.ReadAll(response.Body)

	require := require.New(t)

	require.Equal(200, response.StatusCode)
	require.Equal("", string(responseBody))
}

func TestValidateJWTWithInvalidTokenInQuery(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handlerWithJWTValidation := ValidateJWT(nextHandler, VALID_TOKEN_SECTET)

	request := httptest.NewRequest("GET", "http://testing?token=some_invalid_token", nil)
	responseWriter := httptest.NewRecorder()

	handlerWithJWTValidation.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	responseBody, _ := io.ReadAll(response.Body)

	require := require.New(t)

	require.Equal(403, response.StatusCode)
	require.Equal("Error parsing token\n", string(responseBody))
}

func TestValidateJWTWithValidTokenInQuery(t *testing.T) {
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	handlerWithJWTValidation := ValidateJWT(nextHandler, VALID_TOKEN_SECTET)

	request := httptest.NewRequest("GET", "http://testing?token="+VALID_TOKEN, nil)
	responseWriter := httptest.NewRecorder()

	handlerWithJWTValidation.ServeHTTP(responseWriter, request)

	response := responseWriter.Result()
	responseBody, _ := io.ReadAll(response.Body)

	require := require.New(t)

	require.Equal(200, response.StatusCode)
	require.Equal("", string(responseBody))
}
