package api

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/Miklakapi/chaos-proxy/internal/config"
)

type requestLogContextKey struct{}

type RequestLog struct {
	Method  string
	Path    string
	Status  string
	Chaos   []string
	Started time.Time
}

func NewChaosMiddleware(cfg config.ChaosConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			requestLog := &RequestLog{
				Method:  r.Method,
				Path:    r.URL.RequestURI(),
				Status:  "proxied",
				Chaos:   []string{},
				Started: time.Now(),
			}

			ctx := context.WithValue(r.Context(), requestLogContextKey{}, requestLog)
			r = r.WithContext(ctx)

			defer WriteRequestLog(requestLog)

			if !cfg.Enable {
				handler(w, r)
				return
			}

			if cfg.ConnectionFailure.Enable && cfg.ConnectionFailure.Request.Enable {
				dropped, err := ConnectionFailureHandler(w, cfg.ConnectionFailure.Request)
				if err != nil {
					requestLog.Status = "connection_failure_failed"
					requestLog.Chaos = append(requestLog.Chaos, "connection_failure.request_error")
					log.Printf("connection failure request failed: %v", err)

					handler(w, r)
					return
				}

				if dropped {
					requestLog.Status = "dropped"
					requestLog.Chaos = append(requestLog.Chaos, "connection_failure.request")
					return
				}
			}

			if cfg.Latency.Enable && cfg.Latency.Request.Enable {
				delayed := LatencyHandler(cfg.Latency.Request)
				if delayed {
					requestLog.Status = "delayed"
					requestLog.Chaos = append(requestLog.Chaos, "latency.request")
				}
			}

			handler(w, r)
		}
	}
}

func NewChaosResponseMiddleware(cfg config.ChaosConfig) ResponseMiddleware {
	return func(resp *http.Response) error {
		requestLog := GetRequestLog(resp.Request.Context())

		if !cfg.Enable {
			return nil
		}

		if requestLog != nil && resp != nil {
			requestLog.Status = "proxied"
		}

		return nil
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

func LatencyHandler(cfg config.LatencyPhaseConfig) bool {
	if !ShouldApply(cfg.Probability) {
		return false
	}

	time.Sleep(RandomDurationInRange(cfg.Min, cfg.Max))

	return true
}

func ShouldApply(probability float64) bool {
	return rand.Float64() < probability
}

func RandomDurationInRange(min time.Duration, max time.Duration) time.Duration {
	return time.Duration(rand.Int64N(int64(max-min)+1) + int64(min))
}

func GetRequestLog(ctx context.Context) *RequestLog {
	value := ctx.Value(requestLogContextKey{})
	if value == nil {
		return nil
	}

	requestLog, ok := value.(*RequestLog)
	if !ok {
		return nil
	}

	return requestLog
}

func WriteRequestLog(requestLog *RequestLog) {
	duration := time.Since(requestLog.Started)

	chaosText := "none"
	if len(requestLog.Chaos) > 0 {
		chaosText = strings.Join(requestLog.Chaos, ",")
	}

	message := fmt.Sprintf(
		"request method=%s status=%s duration=%s path=%s chaos=%s",
		requestLog.Method,
		requestLog.Status,
		duration.Round(time.Millisecond),
		requestLog.Path,
		chaosText,
	)

	log.Print(ColorizeRequestLog(message, requestLog))
}

func ColorizeRequestLog(message string, requestLog *RequestLog) string {
	if requestLog.Status == "connection_failure_failed" {
		return "\033[31m" + message + "\033[0m"
	}

	if len(requestLog.Chaos) > 0 {
		return "\033[33m" + message + "\033[0m"
	}

	return message
}
