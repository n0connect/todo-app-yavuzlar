package pow

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/cryptoengine"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/utils"
)

type ChallengeEntry struct {
	Used      bool
	CreatedAt time.Time
	ExpiresAt time.Time
}

type RateLimiter struct {
	tokens         int
	maxTokens      int
	refillTime     time.Time
	refillInterval time.Duration
	lastAccess     time.Time  // Cleanup için
	mutex          sync.Mutex
}

func NewRateLimiter(maxTokens int, refillInterval time.Duration) *RateLimiter {
	now := time.Now()
	return &RateLimiter{
		tokens:         maxTokens,
		maxTokens:      maxTokens,
		refillTime:     now.Add(refillInterval),
		refillInterval: refillInterval,
		lastAccess:     now,
	}
}

func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	rl.lastAccess = now

	// Refill tokens if necessary
	if now.After(rl.refillTime) {
		rl.tokens = rl.maxTokens
		rl.refillTime = now.Add(rl.refillInterval)
	}

	if rl.tokens <= 0 {
		return false
	}

	rl.tokens--
	return true
}

var (
	powLogger = utils.NewLogger("POW")
	// Track issued challenges with expiration
	issuedChallenges = make(map[string]*ChallengeEntry)
	challengeMutex   = sync.RWMutex{}
	// Rate limiter for challenge requests per IP
	challengeRateLimiters = make(map[string]*RateLimiter)
	rateLimiterMutex      = sync.RWMutex{}
	// Cleanup kontrolü için
	cleanupRunning = false
	cleanupMutex   = sync.Mutex{}
)

// GenerateChallenge creates a new proof of work challenge
// ip parameter is used for rate limiting
func GenerateChallenge(ip string) (*models.PoWChallenge, error) {
	powLogger.Info("Generating PoW challenge for IP: %s", ip)

	// Apply rate limiting per IP
	rateLimiterMutex.Lock()
	limiter, exists := challengeRateLimiters[ip]
	if !exists {
		// Create new rate limiter: 1 challenge per 5 minutes
		limiter = NewRateLimiter(1, 5*time.Minute)
		challengeRateLimiters[ip] = limiter
		powLogger.Debug("Created new rate limiter for IP: %s", ip)
	}
	rateLimiterMutex.Unlock()

	if !limiter.Allow() {
		powLogger.Warn("Rate limit exceeded for IP: %s", ip)
		RecordRateLimitHit()
		return nil, fmt.Errorf("rate limit exceeded for IP %s", ip)
	}

	challengeBytes := make([]byte, 16)
	_, err := rand.Read(challengeBytes)
	if err != nil {
		powLogger.LogError("RandomBytes", err)
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	saltBytes := make([]byte, 16)
	_, err = rand.Read(saltBytes)
	if err != nil {
		powLogger.LogError("RandomBytes", err)
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	difficulty := getAdaptiveDifficulty()
	powLogger.Debug("Generated PoW challenge with difficulty: %d for IP: %s", difficulty, ip)

	challengeStr := hex.EncodeToString(challengeBytes)
	saltStr := hex.EncodeToString(saltBytes)

	now := time.Now()
	ttl := int64(300) // 5 minutes

	// Create challenge object
	challengeObj := &models.PoWChallenge{
		Challenge:  challengeStr,
		Timestamp:  now.Unix(),
		Difficulty: difficulty,
		TTL:        ttl,
		Salt:       saltStr,
	}

	// Store the challenge with proper lifecycle management
	challengeKey := challengeStr + saltStr
	challengeMutex.Lock()
	issuedChallenges[challengeKey] = &ChallengeEntry{
		Used:      false,
		CreatedAt: now,
		ExpiresAt: now.Add(time.Duration(ttl) * time.Second),
	}
	challengeMutex.Unlock()

	powLogger.Info("Stored PoW challenge for IP: %s, challenge: %s, expires: %s",
		ip, challengeStr[:8], now.Add(time.Duration(ttl)*time.Second).Format(time.RFC3339))

	RecordChallengeIssued()
	return challengeObj, nil
}

// InitPoW initializes the PoW system and starts cleanup goroutines
func InitPoW() {
	cleanupMutex.Lock()
	defer cleanupMutex.Unlock()

	if cleanupRunning {
		return
	}

	cleanupRunning = true
	powLogger.Info("Starting PoW cleanup goroutine")

	// Challenge cleanup goroutine
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			cleanupExpiredChallenges()
			cleanupInactiveRateLimiters()
		}
	}()
}

// cleanupExpiredChallenges removes expired challenges from the registry
func cleanupExpiredChallenges() {
	challengeMutex.Lock()
	defer challengeMutex.Unlock()

	now := time.Now()
	cleaned := 0

	for key, entry := range issuedChallenges {
		if now.After(entry.ExpiresAt) {
			delete(issuedChallenges, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		powLogger.Debug("Cleaned %d expired challenges, remaining: %d", cleaned, len(issuedChallenges))
	}
}

// cleanupInactiveRateLimiters removes inactive rate limiters
func cleanupInactiveRateLimiters() {
	rateLimiterMutex.Lock()
	defer rateLimiterMutex.Unlock()

	now := time.Now()
	cleaned := 0

	for ip, limiter := range challengeRateLimiters {
		limiter.mutex.Lock()
		// 1 saatten fazla kullanılmamışsa sil
		if now.Sub(limiter.lastAccess) > 1*time.Hour {
			delete(challengeRateLimiters, ip)
			cleaned++
		}
		limiter.mutex.Unlock()
	}

	if cleaned > 0 {
		powLogger.Debug("Cleaned %d inactive rate limiters, remaining: %d", cleaned, len(challengeRateLimiters))
	}
}

// ValidateSolution validates a proof of work solution
func ValidateSolution(solution *models.PoWSolution) bool {
	powLogger.Debug("Validating PoW solution for challenge: %s", solution.Challenge[:8])

	// Atomic validation - lock once, check everything
	challengeKey := solution.Challenge + solution.Salt

	challengeMutex.Lock()
	defer challengeMutex.Unlock()

	// Check if challenge exists
	entry, exists := issuedChallenges[challengeKey]
	if !exists {
		powLogger.Warn("Unknown PoW challenge provided: %s", solution.Challenge[:8])
		RecordValidation(false)
		return false
	}

	// Check if already used (replay attack)
	if entry.Used {
		powLogger.Warn("PoW challenge already used (replay attack): %s", solution.Challenge[:8])
		RecordReplayAttempt()
		RecordValidation(false)
		return false
	}

	// Check expiration
	if time.Now().After(entry.ExpiresAt) {
		powLogger.Warn("PoW solution expired for challenge: %s (expired at: %s)",
			solution.Challenge[:8], entry.ExpiresAt.Format(time.RFC3339))
		delete(issuedChallenges, challengeKey) // Cleanup expired
		RecordExpired()
		RecordValidation(false)
		return false
	}

	// Verify the proof of work using OpenSSL SHA256
	input := []byte(solution.Challenge + solution.Solution + solution.Salt)
	hash, err := cryptoengine.SHA256(input)
	if err != nil {
		powLogger.Error("SHA256 hash computation failed: %v", err)
		RecordValidation(false)
		return false
	}

	result := hasNLeadingZeros(hash, solution.Difficulty)

	if result {
		// Mark as used BEFORE releasing lock
		entry.Used = true
		powLogger.Info("Valid PoW solution accepted for challenge: %s", solution.Challenge[:8])
	} else {
		powLogger.Warn("Invalid PoW solution for challenge: %s (did not meet difficulty requirement)",
			solution.Challenge[:8])
	}

	RecordValidation(result)
	return result
}

// hasNLeadingZeros checks if the hash has at least n leading zero bits
func hasNLeadingZeros(hash []byte, n int) bool {
	if n <= 0 {
		return true
	}

	zeroBits := 0
	
	for _, b := range hash {
		if b == 0 {
			zeroBits += 8
		} else {
			// Count leading zeros in this byte
			for bit := 7; bit >= 0; bit-- {
				if (b>>bit)&1 == 0 {
					zeroBits++
				} else {
					break
				}
			}
			break
		}
		
		if zeroBits >= n {
			break
		}
	}
	
	return zeroBits >= n
}

// getAdaptiveDifficulty returns the current difficulty based on recent activity
func getAdaptiveDifficulty() int {
	// This could be enhanced to check recent registration attempts
	// For now, return a configurable default
	difficulty := config.GetPoWDifficulty()
	if difficulty <= 0 {
		return config.DefaultPoWDifficulty()
	}

	// Clamp to min/max values
	if difficulty < config.MinPoWDifficulty() {
		difficulty = config.MinPoWDifficulty()
	}
	if difficulty > config.MaxPoWDifficulty() {
		difficulty = config.MaxPoWDifficulty()
	}

	return difficulty
}

// GetPoWDifficulty returns the current difficulty level
func GetPoWDifficulty() int {
	return getAdaptiveDifficulty()
}


