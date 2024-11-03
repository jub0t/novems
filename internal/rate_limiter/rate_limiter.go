package rate_limiter

import (
	"time"

	"golang.org/x/time/rate"
)

// CreateLimiter creates a rate limiter based on the config values
func CreateLimiter(rateLimit int) *rate.Limiter {
	return rate.NewLimiter(rate.Limit(rateLimit), rateLimit)
}

// SleepUntilReady blocks until a token is available from the rate limiter
func SleepUntilReady(limiter *rate.Limiter) {
	err := limiter.Wait(nil) // nil context blocks until a token is available
	if err != nil {
		time.Sleep(time.Second) // Fallback sleep in case rate limiter fails
	}
}
