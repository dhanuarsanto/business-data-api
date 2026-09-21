package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"go.internal/business-data-api/pkg/logger"
)

var hideInternalDetails bool

func Init(appEnv string) {
	hideInternalDetails = appEnv != "development"
}

type JSONResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

type CursorPaginationMeta struct {
	HasNextPage bool   `json:"has_next_page"`
	HasPrevPage bool   `json:"has_prev_page"`
	NextCursor  int64  `json:"next_cursor,omitempty"`
}

func Success(w http.ResponseWriter, r *http.Request, data any) {
	write(w, r, http.StatusOK, "API Success", data)
}

func SuccessCreated(w http.ResponseWriter, r *http.Request, data any) {
	write(w, r, http.StatusCreated, "API Created", data)
}

func write(w http.ResponseWriter, r *http.Request, statusCode int, logMsg string, data any) {
	tCtx := logger.GetTraceContext(r.Context())
	slog.Info(logMsg, "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "ip", tCtx.IP, "path", tCtx.Path, "status", statusCode)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(JSONResponse{
		Status: "sukses",
		Data:   data,
	})
}

func Error(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	tCtx := logger.GetTraceContext(r.Context())
	slog.Error("API Error", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "ip", tCtx.IP, "path", tCtx.Path, "status", statusCode, "error", message)

	if hideInternalDetails {
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
