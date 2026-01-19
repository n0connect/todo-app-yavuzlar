//go:build test
// +build test

package middleware

import "sync"

// ResetTrustedProxiesForTesting clears cached trusted proxy configuration.
func ResetTrustedProxiesForTesting() {
	trustedProxyOnce = sync.Once{}
	trustedProxyNets = nil
	trustedProxyIPs = nil
}
