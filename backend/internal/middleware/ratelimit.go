package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

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
}

var globalRateLimiter = &RateLimiter{
	buckets:         make(map[string]*TokenBucket),
	cleanupInterval: 5 * time.Minute,
	lastCleanup:     time.Now(),
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
			// 20 requests per minute = 1 request per 3 seconds
			bucket = NewTokenBucket(20, 3*time.Second)
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
	if len(rl.buckets) > 10000 {
		rl.buckets = make(map[string]*TokenBucket)
		rateLimitLogger.Debug("RateLimiter: cleaned up buckets")
	}
}

// getClientIP extracts client IP from request
// SECURITY: Only trusts X-Real-IP (set by trusted proxy like Nginx)
// X-Forwarded-For is client-controlled and not trusted
func getClientIP(r *http.Request) string {
	// Check X-Real-IP header (set by trusted proxy like Nginx)
	// This is more secure than X-Forwarded-For which is client-controlled
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fallback to RemoteAddr (format: "IP:port")
	// This is the actual connection IP when no proxy is involved
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// RateLimitMiddleware limits requests per IP for login and register endpoints
func RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		bucket := globalRateLimiter.getBucket(ip)

		if !bucket.Allow() {
			rateLimitLogger.Warn("Rate limit exceeded for IP: %s", ip)
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
