//go:build test
// +build test

package auth

// ResetPendingStoreForTesting clears all pending registrations for testing purposes
// SECURITY: This function is ONLY available in test builds (build tag: test)
// It will NOT be compiled into production binaries, preventing security risks
func ResetPendingStoreForTesting() {
	pendingStore.mu.Lock()
	defer pendingStore.mu.Unlock()
	pendingStore.store = make(map[string]*pendingRegistration)
	pendingStoreLogger.Debug("PendingStore: reset for testing")
}
