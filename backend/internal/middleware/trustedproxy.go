package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/utils"
)

var (
	trustedProxyLogger = utils.NewLogger("TRUSTED_PROXY")
	trustedProxyOnce   sync.Once
	trustedProxyNets   []*net.IPNet
	trustedProxyIPs    []net.IP
)

func loadTrustedProxies() {
	trustedProxyOnce.Do(func() {
		raw := config.GetTrustedProxies()
		if raw == "" {
			trustedProxyLogger.Info("Trusted proxies: none configured")
			return
		}

		for _, entry := range strings.Split(raw, ",") {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}

			if ip := net.ParseIP(entry); ip != nil {
				trustedProxyIPs = append(trustedProxyIPs, ip)
				continue
			}

			if _, cidr, err := net.ParseCIDR(entry); err == nil {
				trustedProxyNets = append(trustedProxyNets, cidr)
				continue
			}

			trustedProxyLogger.Warn("Trusted proxy entry ignored (invalid): %s", entry)
		}

		trustedProxyLogger.Info("Trusted proxies loaded: ips=%d cidrs=%d", len(trustedProxyIPs), len(trustedProxyNets))
	})
}

func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}

	loadTrustedProxies()

	for _, trustedIP := range trustedProxyIPs {
		if trustedIP.Equal(ip) {
			return true
		}
	}
	for _, cidr := range trustedProxyNets {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func parseRemoteIP(remoteAddr string) net.IP {
	if remoteAddr == "" {
		return nil
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return net.ParseIP(remoteAddr)
	}
	return net.ParseIP(host)
}

func firstValidIPFromXFF(xff string) net.IP {
	if xff == "" {
		return nil
	}
	for _, part := range strings.Split(xff, ",") {
		ip := net.ParseIP(strings.TrimSpace(part))
		if ip != nil {
			return ip
		}
	}
	return nil
}

// ClientIP returns the real client IP based on trusted proxy configuration.
// Forwarded headers are only trusted when the direct peer is a trusted proxy.
func ClientIP(r *http.Request) string {
	if r == nil {
		return ""
	}

	remoteIP := parseRemoteIP(r.RemoteAddr)
	if isTrustedProxy(remoteIP) {
		if ip := firstValidIPFromXFF(r.Header.Get("X-Forwarded-For")); ip != nil {
			return ip.String()
		}
		if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
			return ip.String()
		}
	}

	if remoteIP != nil {
		return remoteIP.String()
	}
	return ""
}

// IsRequestHTTPS returns true if the request is HTTPS, considering trusted proxy headers.
func IsRequestHTTPS(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	remoteIP := parseRemoteIP(r.RemoteAddr)
	if !isTrustedProxy(remoteIP) {
		return false
	}
	proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))
	return strings.EqualFold(proto, "https")
}
