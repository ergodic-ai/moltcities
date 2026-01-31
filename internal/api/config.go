package api

import (
	"os"
)

// RateLimitConfig holds rate limit values.
type RateLimitConfig struct {
	PixelEditsPerDay     int
	PageUpdatesPerDay    int
	ChannelCreatesPerDay int
	MailSendsPerDay      int
	RegistrationsPerDay  int
}

// DefaultRateLimits returns normal rate limits.
func DefaultRateLimits() RateLimitConfig {
	return RateLimitConfig{
		PixelEditsPerDay:     1,
		PageUpdatesPerDay:    10,
		ChannelCreatesPerDay: 3,
		MailSendsPerDay:      20,
		RegistrationsPerDay:  5,
	}
}

// LiftedRateLimits returns very high rate limits for pre-population.
func LiftedRateLimits() RateLimitConfig {
	return RateLimitConfig{
		PixelEditsPerDay:     10000,
		PageUpdatesPerDay:    10000,
		ChannelCreatesPerDay: 10000,
		MailSendsPerDay:      10000,
		RegistrationsPerDay:  10000,
	}
}

// GetRateLimits returns the current rate limits based on environment.
func GetRateLimits() RateLimitConfig {
	if os.Getenv("LIFT_RATE_LIMITS") == "true" {
		return LiftedRateLimits()
	}
	return DefaultRateLimits()
}

// IsRateLimitLifted returns true if rate limits are temporarily lifted.
func IsRateLimitLifted() bool {
	return os.Getenv("LIFT_RATE_LIMITS") == "true"
}
