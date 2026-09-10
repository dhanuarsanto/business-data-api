package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/pkg/response"

	api_middleware "go.internal/business-data-api/internal/middleware"
)

func SetupRoutes(cfg *config.Config, networkMatrix map[string]bool, roleMatrix map[string][]string, modules ...Module) *chi.Mux {
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
	r.Use(api_middleware.NewRateLimiter(50.0, 100).Middleware())

	r.Get("/docs/swagger.yaml", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "docs/swagger.yaml")
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/swagger.yaml"),
	))

	protected := r.With(
		api_middleware.APIKeyValidator(),
		api_middleware.NetworkRoleGuard(cfg.GlobalLocalOnly, networkMatrix),
	)

	protected.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		response.Success(w, map[string]string{
			"status": "API berjalan dengan normal!",
		})
	})

	for _, m := range modules {
		m.RegisterRoutes(protected.(*chi.Mux), cfg, roleMatrix)
	}

	return r
}
