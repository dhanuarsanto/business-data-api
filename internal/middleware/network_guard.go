package middleware

import (
	"net"
	"net/http"
	"strings"

	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/response"
)

func IsLocalIP(r *http.Request) bool {
	ipStr := r.Header.Get("X-Real-IP")
	if ipStr == "" {
		ipStr = r.Header.Get("X-Forwarded-For")
	}
	if ipStr == "" {
		ipStr, _, _ = net.SplitHostPort(r.RemoteAddr)
	} else {
		ipStr = strings.Split(ipStr, ",")[0]
	}

	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

func NetworkRoleGuard(globalLocalOnly bool, roleMatrix map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			isLocal := IsLocalIP(r)
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
