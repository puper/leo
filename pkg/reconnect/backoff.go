package reconnect

import (
	"math"
	"time"
)

type Backoff struct {
	initialInterval time.Duration
	maxInterval     time.Duration
	multiplier      float64
	attempt         int
}

func NewBackoff(cfg ReconnectConfig) *Backoff {
	return &Backoff{
		initialInterval: cfg.GetInitialInterval(),
		maxInterval:     cfg.GetMaxInterval(),
		multiplier:      cfg.GetMultiplier(),
		attempt:         0,
	}
}

func (b *Backoff) Reset() {
	b.attempt = 0
}

func (b *Backoff) NextDelay() time.Duration {
	if b.attempt == 0 {
		b.attempt = 1
		return b.initialInterval
	}
	// 先限幅再转换，避免超出 time.Duration 范围后出现负延时。
	delay := float64(b.initialInterval) * math.Pow(b.multiplier, float64(b.attempt))
	b.attempt++
	if delay >= float64(b.maxInterval) {
		return b.maxInterval
	}
	return time.Duration(delay)
}

func (b *Backoff) Attempt() int {
	return b.attempt
}
