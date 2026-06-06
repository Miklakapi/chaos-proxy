package api

import (
	"io"
	"net/http"

	"github.com/Miklakapi/chaos-proxy/internal/config"
)

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

type FailingReadCloser struct {
	reader    io.ReadCloser
	bytesLeft int64
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
