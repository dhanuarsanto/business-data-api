package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"

	"go.internal/business-data-api/pkg/jwt"
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
	remoteStr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteStr = r.RemoteAddr
	}
	remote, err := netip.ParseAddr(strings.TrimSpace(remoteStr))
	if err != nil || !remote.IsValid() {
		return netip.Addr{}
	}
	remote = remote.Unmap()

	if t.trust(remote) {
		if ipStr := r.Header.Get("X-Real-IP"); ipStr != "" {
			if ip, err := netip.ParseAddr(strings.TrimSpace(ipStr)); err == nil {
				return ip.Unmap()
			}
		}
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			first := strings.TrimSpace(strings.Split(fwd, ",")[0])
			if ip, err := netip.ParseAddr(first); err == nil {
				return ip.Unmap()
			}
		}
	}
	return remote
}

func (t *TrustedProxyResolver) IsLocal(r *http.Request) bool {
	ip := t.clientAddr(r)
	return ip.IsLoopback() || ip.IsPrivate()
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
				authHeader := r.Header.Get("Authorization")
				if authHeader != "" {
					parts := strings.Split(authHeader, " ")
					if len(parts) == 2 && parts[0] == "Bearer" {
						if parsedClaims, err := jwt.ValidateToken(parts[1]); err == nil {
							claims = parsedClaims
							ok = true
						}
					}
				}
			}

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
