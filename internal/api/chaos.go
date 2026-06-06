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
				dropped, err := ConnectionRequestFailureHandler(w, cfg.ConnectionFailure.Request)
				if err != nil {
					SetRequestLogStatus(requestLog, "chaos_error")
					AddChaosLog(requestLog, "connection_failure.request_error")
					log.Printf("connection failure request failed: %v", err)

					handler(w, r)
					return
				}

				if dropped {
					SetRequestLogStatus(requestLog, "dropped")
					AddChaosLog(requestLog, "connection_failure.request")
					return
				}
			}

			if cfg.Latency.Enable && cfg.Latency.Request.Enable {
				delayed := LatencyHandler(cfg.Latency.Request)
				if delayed {
					AddChaosLog(requestLog, "latency.request")
				}
			}

			if cfg.BandwidthLimit.Enable && cfg.BandwidthLimit.Request.Enable {
				limited := BandwidthLimitRequestHandler(r, cfg.BandwidthLimit.Request)
				if limited {
					AddChaosLog(requestLog, "bandwidth_limit.request")
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

		if cfg.Latency.Enable && cfg.Latency.Response.Enable {
			delayed := LatencyHandler(cfg.Latency.Response)
			if delayed {
				AddChaosLog(requestLog, "latency.response")
			}
		}

		if cfg.BandwidthLimit.Enable && cfg.BandwidthLimit.Response.Enable {
			limited := BandwidthLimitResponseHandler(resp, cfg.BandwidthLimit.Response)
			if limited {
				AddChaosLog(requestLog, "bandwidth_limit.response")
			}
		}

		if cfg.ConnectionFailure.Enable && cfg.ConnectionFailure.Response.Enable {
			dropped := ConnectionResponseFailureHandler(resp, cfg.ConnectionFailure.Response)
			if dropped {
				SetRequestLogStatus(requestLog, "dropped")
				AddChaosLog(requestLog, "connection_failure.response")
				return nil
			}
		}

		return nil
	}
}

func ShouldApply(probability float64) bool {
	return rand.Float64() < probability
}

func RandomBytesInRange(min int64, max int64) int64 {
	return rand.Int64N(max-min+1) + min
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

func AddChaosLog(requestLog *RequestLog, chaos string) {
	if requestLog == nil {
		return
	}

	requestLog.Chaos = append(requestLog.Chaos, chaos)
}

func SetRequestLogStatus(requestLog *RequestLog, status string) {
	if requestLog == nil {
		return
	}

	requestLog.Status = status
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
	if requestLog.Status == "chaos_error" {
		return "\033[31m" + message + "\033[0m"
	}

	if len(requestLog.Chaos) > 0 {
		return "\033[33m" + message + "\033[0m"
	}

	return message
}
