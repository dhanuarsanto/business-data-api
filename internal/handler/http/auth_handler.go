package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/internal/usecase"
	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"

	api_middleware "go.internal/business-data-api/internal/middleware"
)

type AuthHandler struct {
	authUsecase   *usecase.AuthUsecase
	networkMatrix map[string]bool
	ipResolver    *api_middleware.TrustedProxyResolver
	cookieSecure  bool
	authLimiter   *api_middleware.RateLimiter
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = "postgres"
	}
	var payload dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}
	user, err := h.authUsecase.Login(r.Context(), tenant, dbSource, payload.Username, payload.Password)
	if err != nil {
		slog.Warn("Security Alert - Bruteforce / Invalid Login", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "ip", tCtx.IP, "path", tCtx.Path, "username", payload.Username)
		response.Error(w, r, http.StatusUnauthorized, "Username atau password salah")
		return
	}

	if !h.ipResolver.IsLocal(r) && !api_middleware.IsRoleAllowedFromOutside(user.Rules, h.networkMatrix) {
		response.Error(w, r, http.StatusForbidden, "Akses Login ditolak. Peran Anda diwajibkan login melalui jaringan lokal.")
		return
	}

	token, err := jwt.GenerateToken(user.UserID, user.Username, user.Rules, tenant)
	if err != nil {
		response.Error(w, r, http.StatusInternalServerError, "Gagal generate token")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteStrictMode,
	})
	response.Success(w, r, map[string]any{
		"username": user.Username,
		"rules":    user.Rules,
		"token":    token,
	})
}

func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = "postgres"
	}
	var payload dto.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}
	if err := h.authUsecase.CreateUser(r.Context(), tenant, dbSource, domain.User{Username: payload.Username, Password: payload.Password, Rules: payload.Rules}); err != nil {
		if errors.Is(err, usecase.ErrInvalidInput) {
			response.Error(w, r, http.StatusBadRequest, "Username, password, dan rules wajib diisi")
			return
		}
		writeError(w, r, err)
		return
	}
	response.SuccessCreated(w, r, map[string]any{"trace_id": tCtx.TraceID, "message": "Pembuatan user sukses"})
}

func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = "postgres"
	}
	username := chi.URLParam(r, "username")
	var payload dto.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}
	if err := h.authUsecase.UpdateUser(r.Context(), tenant, dbSource, username, payload.Password, payload.Rules); err != nil {
		if errors.Is(err, usecase.ErrInvalidInput) {
			response.Error(w, r, http.StatusBadRequest, "Password dan rules tidak boleh kosong")
			return
		}
		writeError(w, r, err)
		return
	}
	response.Success(w, r, map[string]any{"trace_id": tCtx.TraceID, "message": "Pembaruan user sukses"})
}

func NewAuthHandler(authUsecase *usecase.AuthUsecase, networkMatrix map[string]bool, ipResolver *api_middleware.TrustedProxyResolver, cookieSecure bool) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase, networkMatrix: networkMatrix, ipResolver: ipResolver, cookieSecure: cookieSecure}
}

func (h *AuthHandler) RegisterRoutes(r *chi.Mux, cfg *config.Config, roleMatrix map[string][]string) {
	h.authLimiter = api_middleware.NewRateLimiter(0.2, 5, h.ipResolver)

	r.With(h.authLimiter.Middleware()).Post("/api/v1/{tenant}/auth/login", h.Login)

	r.Route("/api/v1/{tenant}/auth/users", func(users chi.Router) {
		users.Use(api_middleware.RequireToken())
		users.Use(api_middleware.RequireRole(roleMatrix["ManageUsers"]...))
		users.Use(api_middleware.PostgresWriteGuard(cfg.PostgresWriteEnabled))
		users.Post("/", h.CreateUser)
		users.Put("/{username}", h.UpdateUser)
	})
}

func (h *AuthHandler) RateLimiters() []*api_middleware.RateLimiter {
	if h.authLimiter == nil {
		return nil
	}
	return []*api_middleware.RateLimiter{h.authLimiter}
}
