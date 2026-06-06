package api

import (
	"context"
	"fmt"
	"io"
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

type FailingReadCloser struct {
	reader    io.ReadCloser
	bytesLeft int64
}

type ThrottledReadCloser struct {
	reader         io.ReadCloser
	bytesPerSecond int64
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

			if cfg.BandwidthLimit.Enable && cfg.BandwidthLimit.Request.Enable {
				limited := BandwidthLimitRequestHandler(r, cfg.BandwidthLimit.Request)
				if limited {
					requestLog.Status = "limited"
					requestLog.Chaos = append(requestLog.Chaos, "bandwidth_limit.request")
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

		if cfg.ConnectionFailure.Enable && cfg.ConnectionFailure.Response.Enable {
			dropped := ConnectionResponseFailureHandler(resp, cfg.ConnectionFailure.Response)
			if dropped {
				requestLog.Status = "dropped"
				requestLog.Chaos = append(requestLog.Chaos, "connection_failure.response")
				return nil
			}
		}

		if cfg.Latency.Enable && cfg.Latency.Response.Enable {
			delayed := LatencyHandler(cfg.Latency.Response)
			if delayed {
				requestLog.Status = "delayed"
				requestLog.Chaos = append(requestLog.Chaos, "latency.response")
			}
		}

		if cfg.BandwidthLimit.Enable && cfg.BandwidthLimit.Response.Enable {
			limited := BandwidthLimitResponseHandler(resp, cfg.BandwidthLimit.Response)
			if limited {
				requestLog.Status = "limited"
				requestLog.Chaos = append(requestLog.Chaos, "bandwidth_limit.response")
			}
		}

		return nil
	}
}

func ConnectionRequestFailureHandler(w http.ResponseWriter, cfg config.ConnectionFailureRequestConfig) (bool, error) {
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

func ConnectionResponseFailureHandler(r *http.Response, cfg config.ConnectionFailureResponseConfig) bool {
	if !ShouldApply(cfg.Probability) {
		return false
	}

	maxBytes := RandomBytesInRange(cfg.AfterBytesMin, cfg.AfterBytesMax)

	r.Body = NewFailingReadCloser(r.Body, maxBytes)
	r.Close = true

	return true
}

func LatencyHandler(cfg config.LatencyPhaseConfig) bool {
	if !ShouldApply(cfg.Probability) {
		return false
	}

	time.Sleep(RandomDurationInRange(cfg.Min, cfg.Max))

	return true
}

func BandwidthLimitRequestHandler(r *http.Request, cfg config.BandwidthLimitPhaseConfig) bool {
	if !ShouldApply(cfg.Probability) {
		return false
	}

	bytesPerSecond := RandomBytesInRange(cfg.BytesPerSecondMin, cfg.BytesPerSecondMax)

	r.Body = NewThrottledReadCloser(r.Body, bytesPerSecond)
	r.Close = true

	return true
}

func BandwidthLimitResponseHandler(r *http.Response, cfg config.BandwidthLimitPhaseConfig) bool {
	if !ShouldApply(cfg.Probability) {
		return false
	}

	bytesPerSecond := RandomBytesInRange(cfg.BytesPerSecondMin, cfg.BytesPerSecondMax)

	r.Body = NewThrottledReadCloser(r.Body, bytesPerSecond)
	r.Close = true

	return true
}

func ShouldApply(probability float64) bool {
	return rand.Float64() < probability
}

func RandomDurationInRange(min time.Duration, max time.Duration) time.Duration {
	return time.Duration(rand.Int64N(int64(max-min)+1) + int64(min))
}

func RandomBytesInRange(min int64, max int64) int64 {
	return rand.Int64N(max-min+1) + min
}

func NewFailingReadCloser(reader io.ReadCloser, bytesBeforeFailure int64) *FailingReadCloser {
	return &FailingReadCloser{
		reader:    reader,
		bytesLeft: bytesBeforeFailure,
	}
}

func (f *FailingReadCloser) Read(p []byte) (int, error) {
	if f.bytesLeft <= 0 {
		return 0, io.ErrUnexpectedEOF
	}

	if int64(len(p)) > f.bytesLeft {
		p = p[:int(f.bytesLeft)]
	}

	n, err := f.reader.Read(p)
	f.bytesLeft -= int64(n)

	if err != nil {
		return n, err
	}

	if f.bytesLeft <= 0 {
		return n, io.ErrUnexpectedEOF
	}

	return n, nil
}

func (f *FailingReadCloser) Close() error {
	return f.reader.Close()
}

func NewThrottledReadCloser(reader io.ReadCloser, bytesPerSecond int64) *ThrottledReadCloser {
	return &ThrottledReadCloser{
		reader:         reader,
		bytesPerSecond: bytesPerSecond,
	}
}

func (t *ThrottledReadCloser) Read(p []byte) (int, error) {
	n, err := t.reader.Read(p)
	if err != nil {
		return n, err
	}

	if n > 0 {
		duration := time.Duration(float64(n) / float64(t.bytesPerSecond) * float64(time.Second))
		time.Sleep(duration)
	}

	return n, nil
}

func (t *ThrottledReadCloser) Close() error {
	return t.reader.Close()
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
