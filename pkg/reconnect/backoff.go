package reconnect

import "time"

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
	delay := time.Duration(float64(b.initialInterval) * pow(b.multiplier, float64(b.attempt)))
	b.attempt++
	if delay > b.maxInterval {
		return b.maxInterval
	}
	return delay
}

func (b *Backoff) Attempt() int {
	return b.attempt
}

func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}
