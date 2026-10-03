package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"

	"go.internal/business-data-api/pkg/response"
)

type TrustedProxyResolver struct {
	prefixes []netip.Prefix
}

func NewTrustedProxyResolver(prefixes []netip.Prefix) *TrustedProxyResolver {
	return &TrustedProxyResolver{prefixes: prefixes}
}

func (t *TrustedProxyResolver) trust(remote netip.Addr) bool {
	for _, p := range t.prefixes {
		if p.Contains(remote) {
			return true
		}
	}
	return false
}

func (t *TrustedProxyResolver) clientAddr(r *http.Request) netip.Addr {
	remote := parseRemoteAddr(r.RemoteAddr)
	if !remote.IsValid() {
		return netip.Addr{}
	}

	if !t.trust(remote) {
		return remote
	}

	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		chain := strings.Split(fwd, ",")
		for i := len(chain) - 1; i >= 0; i-- {
			ip := parseRemoteAddr(chain[i])
			if !ip.IsValid() {
				break
			}
			if !t.trust(ip) {
				return ip
			}
		}
	}

	if real := parseRemoteAddr(r.Header.Get("X-Real-IP")); real.IsValid() {
		return real
	}

	return remote
}

func parseRemoteAddr(raw string) netip.Addr {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return netip.Addr{}
	}
	if addr, _, err := net.SplitHostPort(raw); err == nil {
		raw = addr
	}
	ip, err := netip.ParseAddr(strings.TrimSpace(raw))
	if err != nil {
		return netip.Addr{}
	}
	return ip.Unmap()
}

func (t *TrustedProxyResolver) IsLocal(r *http.Request) bool {
	ip := t.clientAddr(r)
	return ip.IsLoopback()
}

func NetworkRoleGuard(globalLocalOnly bool, roleMatrix map[string]bool, resolver *TrustedProxyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isLocal := resolver.IsLocal(r)
			if globalLocalOnly {
				if !isLocal {
					response.Error(w, r, http.StatusForbidden, "Akses global ditutup. Hanya menerima koneksi lokal.")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := r.Context().Value(claimsKey).(map[string]any)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			userRole, _ := claims["rules"].(string)

			wajibLokal, roleDefined := roleMatrix[userRole]
			if !roleDefined {
				wajibLokal, _ = roleMatrix["*"]
			}

			if wajibLokal && !isLocal {
				response.Error(w, r, http.StatusForbidden, "Peran Anda hanya diizinkan mengakses modul ini dari jaringan lokal.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func IsRoleAllowedFromOutside(role string, roleMatrix map[string]bool) bool {
	wajibLokal, roleDefined := roleMatrix[role]
	if !roleDefined {
		wajibLokal, _ = roleMatrix["*"]
	}
	return !wajibLokal
}
