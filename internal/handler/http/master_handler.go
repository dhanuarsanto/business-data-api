package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/internal/usecase"
	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"

	api_middleware "go.internal/business-data-api/internal/middleware"
)

type MasterHandler struct {
	usecase *usecase.MasterUsecase
}

func (h *MasterHandler) ListResellerForDropdown(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = "postgres"
	}

	data, err := h.usecase.ListResellerForDropdown(r.Context(), tenant, dbSource)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.Success(w, r, map[string]any{
		"trace_id": tCtx.TraceID,
		"items":    data,
	})
}

func NewMasterHandler(usecase *usecase.MasterUsecase) *MasterHandler {
	return &MasterHandler{usecase: usecase}
}

func (h *MasterHandler) RegisterRoutes(_public, protected chi.Router, cfg *config.Config, roleMatrix map[string][]string) {
	protected.Route("/api/v1/{tenant}/master", func(master chi.Router) {
		master.Use(api_middleware.RequireRole(roleMatrix["ReadResellerDropdown"]...))
		master.Get("/reseller-dropdown", h.ListResellerForDropdown)
	})
}
