package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"todo-app-backend/internal/utils"
)

var (
	pendingStoreLogger = utils.NewLogger("PENDING_STORE")
	pendingStore       = &pendingRegistrationStore{
		store: make(map[string]*pendingRegistration),
	}
	cleanupOnce sync.Once
	cleanupTicker *time.Ticker
)

type pendingRegistration struct {
	AccountNumber string
	ExpiresAt     time.Time
}

type pendingRegistrationStore struct {
	mu    sync.RWMutex
	store map[string]*pendingRegistration
}

// GeneratePendingID generates a unique pending ID
func GeneratePendingID() (string, error) {
	randomBytes := make([]byte, 16) // 128-bit
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes), nil
}

// initCleanup starts the background cleanup goroutine (only once)
// SECURITY: Prevents goroutine leak by starting cleanup only once
func initCleanup() {
	cleanupOnce.Do(func() {
		cleanupTicker = time.NewTicker(1 * time.Minute) // Cleanup every minute
		go func() {
			for range cleanupTicker.C {
				cleanupExpiredPending()
			}
		}()
		pendingStoreLogger.Debug("Started background cleanup goroutine")
	})
}

// StorePendingRegistration stores AccountNumber with pending ID (TTL: 5 minutes)
func StorePendingRegistration(pendingID, accountNumber string) {
	pendingStore.mu.Lock()
	defer pendingStore.mu.Unlock()

	// Start cleanup goroutine if not started (only once)
	initCleanup()

	pendingStore.store[pendingID] = &pendingRegistration{
		AccountNumber: accountNumber,
		ExpiresAt:     time.Now().Add(5 * time.Minute),
	}

	pendingStoreLogger.Debug("Stored pending registration: pendingID=%s (masked)", pendingID[:4]+"..."+pendingID[len(pendingID)-4:])
}

// GetPendingRegistration retrieves AccountNumber by pending ID
// SECURITY: Uses write lock from start to prevent race conditions
// One-time use: deletes entry after retrieval
func GetPendingRegistration(pendingID string) (string, error) {
	pendingStore.mu.Lock() // Write lock (we will delete)
	defer pendingStore.mu.Unlock()

	pending, exists := pendingStore.store[pendingID]
	if !exists {
		return "", ErrInvalidToken
	}

	if time.Now().After(pending.ExpiresAt) {
		// Expired - remove it
		delete(pendingStore.store, pendingID)
		pendingStoreLogger.Debug("Deleted expired pending registration: pendingID=%s (masked)", pendingID[:4]+"..."+pendingID[len(pendingID)-4:])
		return "", ErrExpiredToken
	}

	// One-time use: delete immediately after retrieval
	accountNumber := pending.AccountNumber
	delete(pendingStore.store, pendingID)
	pendingStoreLogger.Debug("Retrieved and deleted pending registration: pendingID=%s (masked)", pendingID[:4]+"..."+pendingID[len(pendingID)-4:])
	return accountNumber, nil
}

// DeletePendingRegistration removes pending registration
func DeletePendingRegistration(pendingID string) {
	pendingStore.mu.Lock()
	defer pendingStore.mu.Unlock()

	delete(pendingStore.store, pendingID)
	pendingStoreLogger.Debug("Deleted pending registration: pendingID=%s (masked)", pendingID[:4]+"..."+pendingID[len(pendingID)-4:])
}

// cleanupExpiredPending removes expired entries (runs in background)
func cleanupExpiredPending() {
	pendingStore.mu.Lock()
	defer pendingStore.mu.Unlock()

	now := time.Now()
	for id, pending := range pendingStore.store {
		if now.After(pending.ExpiresAt) {
			delete(pendingStore.store, id)
		}
	}
}

// StopCleanup stops the background cleanup goroutine gracefully
// This should be called during application shutdown to prevent resource leaks
func StopCleanup() {
	if cleanupTicker != nil {
		cleanupTicker.Stop()
		pendingStoreLogger.Debug("Stopped background cleanup goroutine")
	}
}
