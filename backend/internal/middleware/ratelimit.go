package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/utils"
)

var rateLimitLogger = utils.NewLogger("RATE_LIMIT")

// TokenBucket implements a simple token bucket rate limiter
type TokenBucket struct {
	tokens     int
	maxTokens  int
	refillRate time.Duration
	lastRefill time.Time
	mu         sync.Mutex
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(maxTokens int, refillRate time.Duration) *TokenBucket {
	return &TokenBucket{
		tokens:     maxTokens,
		maxTokens:  maxTokens,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is allowed (consumes one token)
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	// Refill tokens based on elapsed time
	if elapsed > 0 {
		tokensToAdd := int(elapsed / tb.refillRate)
		if tokensToAdd > 0 {
			tb.tokens = tb.tokens + tokensToAdd
			if tb.tokens > tb.maxTokens {
				tb.tokens = tb.maxTokens
			}
			tb.lastRefill = now
		}
	}

	// Check if we have tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// RateLimiter stores token buckets per IP
type RateLimiter struct {
	buckets map[string]*TokenBucket
	mu      sync.RWMutex
	// Cleanup old buckets periodically
	cleanupInterval time.Duration
	lastCleanup     time.Time
	maxTokens       int
	refillInterval  time.Duration
	maxBuckets      int
}

var globalRateLimiter = &RateLimiter{
	buckets: make(map[string]*TokenBucket),
}

func (rl *RateLimiter) loadConfig() error {
	maxTokens := config.GetRateLimitMaxTokens()
	refillSec := config.GetRateLimitRefillIntervalSeconds()
	cleanupSec := config.GetRateLimitCleanupIntervalSeconds()
	maxBuckets := config.GetRateLimitMaxBuckets()
	if maxTokens <= 0 || refillSec <= 0 || cleanupSec <= 0 || maxBuckets <= 0 {
		return fmt.Errorf("rate limit config not set")
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.maxTokens = maxTokens
	rl.refillInterval = time.Duration(refillSec) * time.Second
	rl.cleanupInterval = time.Duration(cleanupSec) * time.Second
	rl.maxBuckets = maxBuckets
	if rl.lastCleanup.IsZero() {
		rl.lastCleanup = time.Now()
	}
	return nil
}

// getBucket gets or creates a token bucket for an IP
func (rl *RateLimiter) getBucket(ip string) *TokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[ip]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		bucket, exists = rl.buckets[ip]
		if !exists {
			bucket = NewTokenBucket(rl.maxTokens, rl.refillInterval)
			rl.buckets[ip] = bucket
		}
		rl.mu.Unlock()
	}

	// SECURITY: Periodic cleanup check (without lock to avoid deadlock)
	// Cleanup is safe to call without lock since it acquires its own lock
	now := time.Now()
	rl.mu.RLock()
	needsCleanup := now.Sub(rl.lastCleanup) > rl.cleanupInterval
	rl.mu.RUnlock()

	if needsCleanup {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		if now.Sub(rl.lastCleanup) > rl.cleanupInterval {
			rl.cleanupUnsafe() // Call cleanup without lock since we already hold it
			rl.lastCleanup = now
		}
		rl.mu.Unlock()
	}

	return bucket
}

// cleanup removes old buckets (simple: remove all, they'll be recreated if needed)
// This function acquires its own lock
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.cleanupUnsafe()
}

// cleanupUnsafe removes old buckets without acquiring a lock
// Caller must hold rl.mu.Lock()
func (rl *RateLimiter) cleanupUnsafe() {
	// Simple cleanup: if we have too many buckets, clear them
	// In production, you might want more sophisticated cleanup
	if len(rl.buckets) > rl.maxBuckets {
		rl.buckets = make(map[string]*TokenBucket)
		rateLimitLogger.Debug("RateLimiter: cleaned up buckets")
	}
}

// RateLimitMiddleware limits requests per IP for login and register endpoints
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := globalRateLimiter.loadConfig(); err != nil {
			rateLimitLogger.LogError("RateLimitConfig", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		ip := ClientIP(r)
		if ip == "" {
			ip = "unknown"
		}
		bucket := globalRateLimiter.getBucket(ip)

		if !bucket.Allow() {
			rateLimitLogger.Warn("Rate limit exceeded for IP: %s", ip)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
