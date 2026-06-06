package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

func New(path string) (Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return cfg, err
	}

	cfg = Normalize(cfg)
	cfg = ApplyDefaults(cfg)

	if err := Validate(cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func Load(path string) (Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func Normalize(cfg Config) Config {
	cfg.Server.Listen = strings.TrimSpace(cfg.Server.Listen)
	cfg.Proxy.Target = strings.TrimSpace(cfg.Proxy.Target)

	return cfg
}

func ApplyDefaults(cfg Config) Config {
	if cfg.Server.ReadTimeout <= 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout <= 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.Server.IdleTimeout <= 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}

	cfg.Chaos.Latency.Request.Probability = NormalizeProbability(cfg.Chaos.Latency.Request.Probability, 0.1)
	if cfg.Chaos.Latency.Request.Min <= 0 {
		cfg.Chaos.Latency.Request.Min = 200 * time.Millisecond
	}
	if cfg.Chaos.Latency.Request.Max <= 0 {
		cfg.Chaos.Latency.Request.Max = 2000 * time.Millisecond
	}

	cfg.Chaos.Latency.Response.Probability = NormalizeProbability(cfg.Chaos.Latency.Response.Probability, 0.1)
	if cfg.Chaos.Latency.Response.Min <= 0 {
		cfg.Chaos.Latency.Response.Min = 200 * time.Millisecond
	}
	if cfg.Chaos.Latency.Response.Max <= 0 {
		cfg.Chaos.Latency.Response.Max = 2000 * time.Millisecond
	}

	cfg.Chaos.ConnectionFailure.Request.Probability = NormalizeProbability(cfg.Chaos.ConnectionFailure.Request.Probability, 0.02)

	cfg.Chaos.ConnectionFailure.Response.Probability = NormalizeProbability(cfg.Chaos.ConnectionFailure.Response.Probability, 0.02)
	if cfg.Chaos.ConnectionFailure.Response.AfterBytesMin <= 0 {
		cfg.Chaos.ConnectionFailure.Response.AfterBytesMin = 1024
	}
	if cfg.Chaos.ConnectionFailure.Response.AfterBytesMax <= 0 {
		cfg.Chaos.ConnectionFailure.Response.AfterBytesMax = 65536
	}

	cfg.Chaos.BandwidthLimit.Request.Probability = NormalizeProbability(cfg.Chaos.BandwidthLimit.Request.Probability, 0.1)
	if cfg.Chaos.BandwidthLimit.Request.BytesPerSecondMin <= 0 {
		cfg.Chaos.BandwidthLimit.Request.BytesPerSecondMin = 10240
	}
	if cfg.Chaos.BandwidthLimit.Request.BytesPerSecondMax <= 0 {
		cfg.Chaos.BandwidthLimit.Request.BytesPerSecondMax = 20480
	}

	cfg.Chaos.BandwidthLimit.Response.Probability = NormalizeProbability(cfg.Chaos.BandwidthLimit.Response.Probability, 0.1)
	if cfg.Chaos.BandwidthLimit.Response.BytesPerSecondMin <= 0 {
		cfg.Chaos.BandwidthLimit.Response.BytesPerSecondMin = 10240
	}
	if cfg.Chaos.BandwidthLimit.Response.BytesPerSecondMax <= 0 {
		cfg.Chaos.BandwidthLimit.Response.BytesPerSecondMax = 20480
	}

	return cfg
}

func NormalizeProbability(probability float64, defaultProbability float64) float64 {
	if probability <= 0 {
		return defaultProbability
	}

	if probability > 1 {
		return 1
	}

	return probability
}

func Validate(cfg Config) error {
	var validationErrors []error

	if cfg.Server.Listen == "" {
		validationErrors = append(validationErrors, errors.New("server.listen is required"))
	} else {
		_, port, err := net.SplitHostPort(cfg.Server.Listen)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("server.listen must be in host:port format: %w", err))
		} else {
			portNumber, err := strconv.Atoi(port)
			if err != nil {
				validationErrors = append(validationErrors, errors.New("server.listen port must be a number"))
			} else if portNumber < 1 || portNumber > 65535 {
				validationErrors = append(validationErrors, errors.New("server.listen port must be between 1 and 65535"))
			}
		}
	}

	if cfg.Proxy.Target == "" {
		validationErrors = append(validationErrors, errors.New("proxy.target is required"))
	} else {
		targetURL, err := url.Parse(cfg.Proxy.Target)
		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("proxy.target is invalid: %w", err))
		} else {
			if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
				validationErrors = append(validationErrors, errors.New("proxy.target scheme must be http or https"))
			}

			if targetURL.Host == "" {
				validationErrors = append(validationErrors, errors.New("proxy.target host is required"))
			}
		}
	}

	if cfg.Chaos.Latency.Request.Min > cfg.Chaos.Latency.Request.Max {
		validationErrors = append(validationErrors, errors.New("chaos.latency.request.min must be less than or equal to chaos.latency.request.max"))
	}

	if cfg.Chaos.Latency.Response.Min > cfg.Chaos.Latency.Response.Max {
		validationErrors = append(validationErrors, errors.New("chaos.latency.response.min must be less than or equal to chaos.latency.response.max"))
	}

	if cfg.Chaos.ConnectionFailure.Response.AfterBytesMin > cfg.Chaos.ConnectionFailure.Response.AfterBytesMax {
		validationErrors = append(validationErrors, errors.New("chaos.connection_failure.response.after_bytes_min must be less than or equal to chaos.connection_failure.response.after_bytes_max"))
	}

	if cfg.Chaos.BandwidthLimit.Request.BytesPerSecondMin > cfg.Chaos.BandwidthLimit.Request.BytesPerSecondMax {
		validationErrors = append(validationErrors, errors.New("chaos.bandwidth_limit.request.bytes_per_second_min must be less than or equal to chaos.bandwidth_limit.request.bytes_per_second_max"))
	}

	if cfg.Chaos.BandwidthLimit.Response.BytesPerSecondMin > cfg.Chaos.BandwidthLimit.Response.BytesPerSecondMax {
		validationErrors = append(validationErrors, errors.New("chaos.bandwidth_limit.response.bytes_per_second_min must be less than or equal to chaos.bandwidth_limit.response.bytes_per_second_max"))
	}

	return errors.Join(validationErrors...)
}
