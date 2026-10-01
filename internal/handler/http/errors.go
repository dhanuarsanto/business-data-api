package http

import (
	"errors"
	"log/slog"
	"net/http"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/pkg/database"
	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"
)

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, database.ErrTenantNotFound):
		response.Error(w, r, http.StatusNotFound, "Tenant tidak ditemukan")
	case errors.Is(err, domain.ErrSourceNotValid):
		response.Error(w, r, http.StatusBadRequest, "Header X-DB-Source tidak valid")
	default:
		tCtx := logger.GetTraceContext(r.Context())
		slog.Error("Kesalahan internal tak Tertangani",
			"trace_id", tCtx.TraceID,
			"developer", tCtx.Developer,
			"path", tCtx.Path,
			"method", r.Method,
			"error", err,
		)
		response.Error(w, r, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}
