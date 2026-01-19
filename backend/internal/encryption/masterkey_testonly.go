//go:build test
// +build test

package encryption

import "sync"

// ResetMasterKeysForTesting clears cached master key state for tests.
func ResetMasterKeysForTesting() {
	masterKeyOnce = sync.Once{}
	masterKeyError = nil
	masterKeys = masterKeySet{}
}
