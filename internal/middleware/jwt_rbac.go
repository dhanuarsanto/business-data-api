package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"
)

type contextKey string

const claimsKey contextKey = "user_claims"

func RequireToken() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, r, http.StatusUnauthorized, "Token otorisasi diperlukan")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				response.Error(w, r, http.StatusUnauthorized, "Format token tidak valid")
				return
			}

			claims, err := jwt.ValidateToken(parts[1])
			if err != nil {
				response.Error(w, r, http.StatusUnauthorized, "Token kadaluarsa atau tidak valid")
				return
			}

			if userID, ok := claims["user_id"].(float64); ok {
				tCtx := logger.GetTraceContext(r.Context())
				tCtx.UserID = int(userID)
			}

			tenant := chi.URLParam(r, "tenant")
			if claimTenant, ok := claims["tenant"].(string); !ok || claimTenant == "" || claimTenant != tenant {
				response.Error(w, r, http.StatusForbidden, "Token tidak berlaku untuk tenant ini")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, map[string]any(claims))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(claimsKey).(map[string]any)
			if !ok {
				tCtx := logger.GetTraceContext(r.Context())
				slog.Warn("Security Alert - RBAC Block", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "ip", tCtx.IP, "path", tCtx.Path, "role", claims["rules"])
				response.Error(w, r, http.StatusForbidden, "Gagal mengidentifikasi role")
				return
			}

			userRole, ok := claims["rules"].(string)
			if !ok {
				tCtx := logger.GetTraceContext(r.Context())
				slog.Warn("Security Alert - RBAC Block", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "ip", tCtx.IP, "path", tCtx.Path, "role", claims["rules"])
				response.Error(w, r, http.StatusForbidden, "Role tidak ditemukan pada token")
				return
			}

			if !slices.Contains(allowedRoles, userRole) {
				tCtx := logger.GetTraceContext(r.Context())
				slog.Warn("Security Alert - RBAC Block", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "ip", tCtx.IP, "path", tCtx.Path, "role", claims["rules"])
				response.Error(w, r, http.StatusForbidden, "Anda tidak memiliki akses ke resource ini")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
