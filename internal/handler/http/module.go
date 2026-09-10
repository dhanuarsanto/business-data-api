package http

import (
	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/internal/config"
)

type Module interface {
	RegisterRoutes(r *chi.Mux, cfg *config.Config, roleMatrix map[string][]string)
}
