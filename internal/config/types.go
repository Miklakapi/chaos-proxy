package config

import "time"

type Config struct {
	Server ServerConfig `yaml:"server"`
	Proxy  ProxyConfig  `yaml:"proxy"`
	Chaos  ChaosConfig  `yaml:"chaos"`
}

type ServerConfig struct {
	Listen       string        `yaml:"listen"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type ProxyConfig struct {
	Target       string            `yaml:"target"`
	PreserveHost bool              `yaml:"preserve_host"`
	AddHeaders   map[string]string `yaml:"add_headers"`
}

type ChaosConfig struct {
	Enable            bool                    `yaml:"enable"`
	Latency           LatencyConfig           `yaml:"latency"`
	ConnectionFailure ConnectionFailureConfig `yaml:"connection_failure"`
	BandwidthLimit    BandwidthLimitConfig    `yaml:"bandwidth_limit"`
}

type LatencyConfig struct {
	Enable   bool               `yaml:"enable"`
	Request  LatencyPhaseConfig `yaml:"request"`
	Response LatencyPhaseConfig `yaml:"response"`
}

type LatencyPhaseConfig struct {
	Enable      bool          `yaml:"enable"`
	Probability float64       `yaml:"probability"`
	Min         time.Duration `yaml:"min"`
	Max         time.Duration `yaml:"max"`
}

type ConnectionFailureConfig struct {
	Enable   bool                            `yaml:"enable"`
	Request  ConnectionFailureRequestConfig  `yaml:"request"`
	Response ConnectionFailureResponseConfig `yaml:"response"`
}

type ConnectionFailureRequestConfig struct {
	Enable      bool    `yaml:"enable"`
	Probability float64 `yaml:"probability"`
}

type ConnectionFailureResponseConfig struct {
	Enable        bool    `yaml:"enable"`
	Probability   float64 `yaml:"probability"`
	AfterBytesMin int64   `yaml:"after_bytes_min"`
	AfterBytesMax int64   `yaml:"after_bytes_max"`
}

type BandwidthLimitConfig struct {
	Enable   bool                      `yaml:"enable"`
	Request  BandwidthLimitPhaseConfig `yaml:"request"`
	Response BandwidthLimitPhaseConfig `yaml:"response"`
}

type BandwidthLimitPhaseConfig struct {
	Enable            bool    `yaml:"enable"`
	Probability       float64 `yaml:"probability"`
	BytesPerSecondMin int64   `yaml:"bytes_per_second_min"`
	BytesPerSecondMax int64   `yaml:"bytes_per_second_max"`
}
