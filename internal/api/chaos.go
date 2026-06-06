package api

import (
	"math/rand/v2"
	"net/http"

	"github.com/Miklakapi/chaos-proxy/internal/config"
)

func NewChaosMiddleware(cfg config.ChaosConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !cfg.Enable {
				handler(w, r)
				return
			}
			if cfg.ConnectionFailure.Enable && cfg.ConnectionFailure.Request.Enable {
				dropped, err := ConnectionFailureHandler(w, cfg.ConnectionFailure.Request)
				if err != nil {
					//
				}

				if dropped {
					return
				}
			}
			handler(w, r)
		}
	}
}

func ConnectionFailureHandler(w http.ResponseWriter, cfg config.ConnectionFailureRequestConfig) (bool, error) {
	if !ShouldApply(cfg.Probability) {
		return false, nil
	}

	controller := http.NewResponseController(w)

	conn, _, err := controller.Hijack()
	if err != nil {
		return false, err
	}

	if err := conn.Close(); err != nil {
		return false, err
	}

	return true, nil
}

func ShouldApply(probability float64) bool {
	return rand.Float64() < probability
}

func NewChaosResponseMiddleware(cfg config.ChaosConfig) ResponseMiddleware {
	return func(r *http.Response) error {
		return nil
	}
}
