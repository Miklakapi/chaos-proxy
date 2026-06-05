package api

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/Miklakapi/chaos-proxy/internal/config"
)

func ChaosProxyHandler(cfg config.ProxyConfig) http.HandlerFunc {
	target, _ := url.Parse(cfg.Target)
	proxy := httputil.NewSingleHostReverseProxy(target)

	return func(w http.ResponseWriter, r *http.Request) {
		if !cfg.PreserveHost {
			r.Host = target.Host
		}

		for key, value := range cfg.AddHeaders {
			r.Header.Set(key, value)
		}

		proxy.ServeHTTP(w, r)
	}
}
