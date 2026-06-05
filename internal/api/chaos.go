package api

import (
	"net/http"

	"github.com/Miklakapi/chaos-proxy/internal/config"
)

func NewChaosMiddleware(cfg config.ChaosConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// chaos
			handler(w, r)
		}
	}
}

func NewChaosResponseMiddleware(cfg config.ChaosConfig) ResponseMiddleware {
	return func(r *http.Response) error {
		return nil
	}
}
