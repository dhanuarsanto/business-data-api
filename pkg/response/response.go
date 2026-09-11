package response

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"go.internal/business-data-api/pkg/logger"
)

type JSONResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type CursorPaginationMeta struct {
	TotalData   int   `json:"total_data"`
	HasNextPage bool  `json:"has_next_page"`
	HasPrevPage bool  `json:"has_prev_page"`
	NextCursor  int64 `json:"next_cursor,omitempty"`
}

func Success(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(JSONResponse{
		Status: "sukses",
		Data:   data,
	})
}

func Error(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	tCtx := logger.GetTraceContext(r.Context())
	slog.Error("API Error", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "status", statusCode, "error", message, "path", tCtx.Path)

	env := os.Getenv("APP_ENV")
	if env != "development" {
		switch {
		case statusCode >= 500:
			message = "Terjadi kesalahan internal pada server"
		case statusCode == http.StatusUnauthorized:
			message = "Akses ditolak"
		case statusCode == http.StatusForbidden:
			message = "Akses ditolak"
		case statusCode == http.StatusNotFound:
			message = "Resource tidak ditemukan"
		case statusCode == http.StatusBadRequest:
			message = "Permintaan tidak valid"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(JSONResponse{
		Status:  "gagal",
		Message: message,
	})
}
