package middleware

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.internal/business-data-api/pkg/logger"
)

func SecurityTracer(resolver *TrustedProxyResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			b := make([]byte, 16)
			rand.Read(b)
			traceID := fmt.Sprintf("TRC-%x", b)

			ip := r.RemoteAddr
			if resolver != nil {
				if addr := resolver.clientAddr(r); addr.IsValid() {
					ip = addr.String()
				}
			}

			tCtx := &logger.TraceContext{
				TraceID:   traceID,
				Developer: "Tidak Diketahui",
				IP:        ip,
				Path:      r.URL.Path,
			}
			ctx := logger.SetTraceContext(r.Context(), tCtx)

			next.ServeHTTP(ww, r.WithContext(ctx))

			slog.Info("Audit Log",
				"trace_id", tCtx.TraceID,
				"method", r.Method,
				"path", tCtx.Path,
				"ip", tCtx.IP,
				"developer", tCtx.Developer,
				"status", ww.Status(),
				"latency_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
