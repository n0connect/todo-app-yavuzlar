package pow

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/models"
)

func TestChallengeLifecycle(t *testing.T) {
	InitPoW()

	// Generate challenge
	challenge, err := GenerateChallenge("192.168.1.1")
	if err != nil {
		t.Fatalf("Failed to generate challenge: %v", err)
	}

	// Verify challenge properties
	if challenge.Challenge == "" {
		t.Fatal("Challenge string is empty")
	}

	if challenge.Salt == "" {
		t.Fatal("Salt is empty")
	}

	// Check that difficulty is within expected range (with new default of 22)
	if challenge.Difficulty < config.MinPoWDifficulty() || challenge.Difficulty > config.MaxPoWDifficulty() {
		t.Fatalf("Difficulty out of expected range [%d-%d]: %d",
			config.MinPoWDifficulty(), config.MaxPoWDifficulty(), challenge.Difficulty)
	}

	// Verify challenge is stored
	key := challenge.Challenge + challenge.Salt
	challengeMutex.RLock()
	entry, exists := issuedChallenges[key]
	challengeMutex.RUnlock()

	if !exists {
		t.Fatal("Challenge not stored")
	}

	if entry.Used {
		t.Fatal("New challenge marked as used")
	}

	if time.Now().After(entry.ExpiresAt) {
		t.Fatal("Challenge already expired")
	}
}

func TestRaceCondition(t *testing.T) {
	InitPoW()

	// Generate a challenge with a unique IP to avoid rate limiting
	ip := "192.168.2.1" // Different IP to avoid rate limit
	challenge, err := GenerateChallenge(ip)
	if err != nil {
		t.Fatalf("Failed to generate challenge: %v", err)
	}

	// Create a solution that will fail the PoW validation but pass the challenge existence check
	solution := &models.PoWSolution{
		Challenge:  challenge.Challenge,
		Solution:   "invalid_solution", // This will fail the PoW check
		Salt:       challenge.Salt,
		Difficulty: challenge.Difficulty,
		Timestamp:  challenge.Timestamp,
		TTL:        challenge.TTL,
	}

	var wg sync.WaitGroup
	attemptCount := 0
	mutex := sync.Mutex{}

	// 100 goroutines trying to validate the same solution simultaneously
	// This tests the race condition in the challenge tracking logic
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// This should be safe even with multiple concurrent calls
			_ = ValidateSolution(solution)
			mutex.Lock()
			attemptCount++
			mutex.Unlock()
		}()
	}

	wg.Wait()

	if attemptCount != 100 {
		t.Fatalf("Expected 100 attempts, got %d", attemptCount)
	}

	// Check that the challenge still exists and is not marked as used
	challengeKey := challenge.Challenge + challenge.Salt
	challengeMutex.RLock()
	entry, exists := issuedChallenges[challengeKey]
	if !exists {
		t.Fatal("Challenge was deleted during race condition test")
	}
	if entry.Used {
		t.Fatal("Challenge was marked as used during race condition test")
	}
	challengeMutex.RUnlock()
}

func TestMemoryCleanup(t *testing.T) {
	InitPoW()
	
	// Clear existing challenges for clean test
	challengeMutex.Lock()
	issuedChallenges = make(map[string]*ChallengeEntry)
	challengeMutex.Unlock()
	
	// Generate expired challenges
	for i := 0; i < 100; i++ {
		challengeBytes := make([]byte, 16)
		saltBytes := make([]byte, 16)
		rand.Read(challengeBytes)
		rand.Read(saltBytes)
		
		key := hex.EncodeToString(challengeBytes) + hex.EncodeToString(saltBytes)
		
		challengeMutex.Lock()
		issuedChallenges[key] = &ChallengeEntry{
			Used:      false,
			CreatedAt: time.Now().Add(-10 * time.Minute),
			ExpiresAt: time.Now().Add(-5 * time.Minute), // Expired
		}
		challengeMutex.Unlock()
	}
	
	initialCount := len(issuedChallenges)
	
	// Run cleanup
	cleanupExpiredChallenges()
	
	finalCount := len(issuedChallenges)
	
	if finalCount >= initialCount {
		t.Fatalf("Cleanup didn't work: before=%d, after=%d", initialCount, finalCount)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	InitPoW()
	
	// Clear existing rate limiters for clean test
	rateLimiterMutex.Lock()
	challengeRateLimiters = make(map[string]*RateLimiter)
	rateLimiterMutex.Unlock()
	
	// Create inactive rate limiters
	for i := 0; i < 10; i++ {
		ip := "192.168.1." + string(rune('0'+i))
		limiter := NewRateLimiter(1, 1*time.Minute)
		// Simulate last access 2 hours ago
		limiter.lastAccess = time.Now().Add(-2 * time.Hour)
		
		rateLimiterMutex.Lock()
		challengeRateLimiters[ip] = limiter
		rateLimiterMutex.Unlock()
	}
	
	initialCount := len(challengeRateLimiters)
	
	// Run cleanup
	cleanupInactiveRateLimiters()
	
	finalCount := len(challengeRateLimiters)
	
	if finalCount >= initialCount {
		t.Fatalf("Rate limiter cleanup didn't work: before=%d, after=%d", initialCount, finalCount)
	}
}

func TestValidateSolutionWithExpiredChallenge(t *testing.T) {
	InitPoW()
	
	// Create an expired challenge entry manually for testing
	challengeStr := "testchallenge1234567890abcdef"
	saltStr := "testsalt1234567890abcdef"
	
	key := challengeStr + saltStr
	expiredTime := time.Now().Add(-10 * time.Minute) // Already expired
	
	challengeMutex.Lock()
	issuedChallenges[key] = &ChallengeEntry{
		Used:      false,
		CreatedAt: time.Now().Add(-15 * time.Minute),
		ExpiresAt: expiredTime,
	}
	challengeMutex.Unlock()
	
	// Create a solution for the expired challenge
	solution := &models.PoWSolution{
		Challenge:  challengeStr,
		Solution:   "0000", // Mock solution
		Salt:       saltStr,
		Difficulty: 20,
		Timestamp:  time.Now().Unix(),
		TTL:        300,
	}
	
	// Validate should fail due to expiration
	result := ValidateSolution(solution)
	if result {
		t.Fatal("Validation should have failed for expired challenge")
	}
	
	// Check that the expired challenge was cleaned up
	challengeMutex.RLock()
	_, exists := issuedChallenges[key]
	challengeMutex.RUnlock()
	
	if exists {
		t.Fatal("Expired challenge was not cleaned up")
	}
}

func TestValidateSolutionWithUsedChallenge(t *testing.T) {
	InitPoW()
	
	// Create a used challenge entry manually for testing
	challengeStr := "testchallenge1234567890abcdeg"
	saltStr := "testsalt1234567890abcdefh"
	
	key := challengeStr + saltStr
	
	challengeMutex.Lock()
	issuedChallenges[key] = &ChallengeEntry{
		Used:      true, // Already used
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	challengeMutex.Unlock()
	
	// Create a solution for the used challenge
	solution := &models.PoWSolution{
		Challenge:  challengeStr,
		Solution:   "0000", // Mock solution
		Salt:       saltStr,
		Difficulty: 20,
		Timestamp:  time.Now().Unix(),
		TTL:        300,
	}
	
	// Validate should fail due to being already used
	result := ValidateSolution(solution)
	if result {
		t.Fatal("Validation should have failed for already used challenge")
	}
}