package api

import (
	"math/rand/v2"
	"time"

	"github.com/Miklakapi/chaos-proxy/internal/config"
)

func LatencyHandler(cfg config.LatencyPhaseConfig) bool {
	if !ShouldApply(cfg.Probability) {
		return false
	}

	time.Sleep(RandomDurationInRange(cfg.Min, cfg.Max))

	return true
}

func RandomDurationInRange(min time.Duration, max time.Duration) time.Duration {
	return time.Duration(rand.Int64N(int64(max-min)+1) + int64(min))
}
