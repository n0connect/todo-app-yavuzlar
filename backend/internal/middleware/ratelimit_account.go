package middleware

// AllowAccountLookup applies rate limiting for a specific account lookup hash.
func AllowAccountLookup(lookup string) bool {
	if err := globalRateLimiter.loadConfig(); err != nil {
		rateLimitLogger.LogError("RateLimitConfig", err)
		return false
	}
	if lookup == "" {
		return true
	}
	bucket := globalRateLimiter.getBucket("acct:" + lookup)
	return bucket.Allow()
}
