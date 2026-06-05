package api

import (
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
