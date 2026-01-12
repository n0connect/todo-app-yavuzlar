//go:build test
// +build test

package middleware

import "time"

// ResetRateLimiterForTesting clears all buckets for testing purposes
// SECURITY: This function is ONLY available in test builds (build tag: test)
// It will NOT be compiled into production binaries, preventing security risks
func ResetRateLimiterForTesting() {
	globalRateLimiter.mu.Lock()
	defer globalRateLimiter.mu.Unlock()
	globalRateLimiter.buckets = make(map[string]*TokenBucket)
	globalRateLimiter.lastCleanup = time.Now()
	rateLimitLogger.Debug("RateLimiter: reset for testing")
}
