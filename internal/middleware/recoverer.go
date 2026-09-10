package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"
)

func PanicRecoverer() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					tCtx := logger.GetTraceContext(r.Context())
					stack := string(debug.Stack())
					slog.Error("System Panic / Crash", "trace_id", tCtx.TraceID, "error", fmt.Sprintf("%v", err), "stack", stack)
					response.Error(w, r, http.StatusInternalServerError, "Terjadi kesalahan sistem yang fatal")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
