package api

import (
	"log"
	"net/http"
	"time"
)

/*
Logging Middleware
This is responsible for logging all the HTTP requests
received/listened by the server

example:
POST /health 2.033ms

It accepts a handler function in parameters,
performs middleware logic, and servers the handler function
and returns the handler wrapped with middleware logic
*/
func Logging(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)
		log.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}
