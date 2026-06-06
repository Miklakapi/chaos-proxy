package api

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Miklakapi/chaos-proxy/internal/config"
)

type ResponseMiddleware func(resp *http.Response) error

func NewChaosProxyHandler(cfg config.ProxyConfig, responseMiddleware ResponseMiddleware) (http.HandlerFunc, error) {
	target, err := url.Parse(cfg.Target)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	if responseMiddleware != nil {
		proxy.ModifyResponse = responseMiddleware
	}

	proxy.ErrorHandler = HandleProxyError

	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.PreserveHost {
			r.Host = target.Host
		}

		for key, value := range cfg.AddHeaders {
			r.Header.Set(key, value)
		}

		proxy.ServeHTTP(w, r)
	}, nil
}

func HandleProxyError(w http.ResponseWriter, r *http.Request, err error) {
	requestLog := GetRequestLog(r.Context())
	SetRequestLogStatus(requestLog, "proxy_error")

	log.Printf("proxy error method=%s path=%s error=%v", r.Method, r.URL.RequestURI(), err)

	http.Error(w, "bad gateway", http.StatusBadGateway)
}
