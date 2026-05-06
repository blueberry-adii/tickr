package api

import (
	"log"
	"net/http"
	"time"
)

/*
This is responsible for logging all the HTTP requests
received/listened by the server
*/
func Logging(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)
		log.Printf("%s %s %s\n", r.Method, r.URL.Path, time.Since(start))
	})
}
