package api

import (
	"io"
	"net/http"
	"time"

	"github.com/Miklakapi/chaos-proxy/internal/config"
)

func BandwidthLimitRequestHandler(r *http.Request, cfg config.BandwidthLimitPhaseConfig) bool {
	if !ShouldApply(cfg.Probability) {
		return false
	}

	bytesPerSecond := RandomBytesInRange(cfg.BytesPerSecondMin, cfg.BytesPerSecondMax)

	r.Body = NewThrottledReadCloser(r.Body, bytesPerSecond)

	return true
}

func BandwidthLimitResponseHandler(r *http.Response, cfg config.BandwidthLimitPhaseConfig) bool {
	if !ShouldApply(cfg.Probability) {
		return false
	}

	bytesPerSecond := RandomBytesInRange(cfg.BytesPerSecondMin, cfg.BytesPerSecondMax)

	r.Body = NewThrottledReadCloser(r.Body, bytesPerSecond)

	return true
}

type ThrottledReadCloser struct {
	reader         io.ReadCloser
	bytesPerSecond int64
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
