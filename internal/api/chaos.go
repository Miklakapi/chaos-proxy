package api

import "net/http"

func ChaosMiddleware(handler http.HandlerFunc) http.HandlerFunc {
	//
	return func(w http.ResponseWriter, r *http.Request) {
		// chaos before
		handler(w, r)
		// chaos after
	}
}
