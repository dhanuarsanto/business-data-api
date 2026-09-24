package http

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/pkg/response"

	api_middleware "go.internal/business-data-api/internal/middleware"
)

func swaggerYAMLPath() string {
	candidates := []string{"docs/swagger.yaml"}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, "docs", "swagger.yaml"),
			filepath.Join(dir, "..", "docs", "swagger.yaml"),
		)
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
			return p
		}
	}
	return candidates[0]
}

func SetupRoutes(cfg *config.Config, resolver *api_middleware.TrustedProxyResolver, networkMatrix map[string]bool, roleMatrix map[string][]string, modules ...Module) (*chi.Mux, []*api_middleware.RateLimiter, *api_middleware.KeyManager) {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.GetAllowedOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-KEY", "X-DB-Source"},
		ExposedHeaders:   []string{"Link", "Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Use(api_middleware.PanicRecoverer())
	r.Use(api_middleware.SecurityTracer())
	r.Use(api_middleware.RequestBodyLimit(cfg.MaxBodyBytes))

	r.Get("/docs/swagger.yaml", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, swaggerYAMLPath())
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.yaml"),
	))

	keyPath := cfg.APIKeysPath
	if keyPath == "" {
		keyPath = "api_keys.json"
	}
	keyManager := api_middleware.NewKeyManager(keyPath)

	var limiters []*api_middleware.RateLimiter
	globalLimiter := api_middleware.NewRateLimiter(50.0, 100, resolver)
	limiters = append(limiters, globalLimiter)

	protectedApiKey := r.With(
		keyManager.Middleware(),
		globalLimiter.Middleware(),
	)

	protectedApiKey.NotFound(func(w http.ResponseWriter, req *http.Request) {
		response.Error(w, req, http.StatusNotFound, "Endpoint tidak ditemukan")
	})

	protectedApiKey.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		response.Error(w, req, http.StatusMethodNotAllowed, "Method HTTP tidak diizinkan pada endpoint ini")
	})

	public := protectedApiKey.With(
		api_middleware.NetworkRoleGuard(cfg.GlobalLocalOnly, networkMatrix, resolver),
	)

	protected := protectedApiKey.With(
		api_middleware.RequireToken(),
		api_middleware.NetworkRoleGuard(cfg.GlobalLocalOnly, networkMatrix, resolver),
	)

	public.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		response.Success(w, req, map[string]string{
			"status": "API berjalan dengan normal!",
		})
	})

	for _, m := range modules {
		m.RegisterRoutes(public, protected, cfg, roleMatrix)
		if limiterProvider, ok := m.(interface {
			RateLimiters() []*api_middleware.RateLimiter
		}); ok {
			limiters = append(limiters, limiterProvider.RateLimiters()...)
		}
	}

	return r, limiters, keyManager
}
