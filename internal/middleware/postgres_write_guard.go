package middleware

import (
	"net/http"

	"go.internal/business-data-api/pkg/response"
)

func PostgresWriteGuard(enabled bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
				dbSource := r.Header.Get("X-DB-Source")
				if dbSource == "" {
					dbSource = "postgres"
				}
				if dbSource == "postgres" && !enabled {
					response.Error(w, r, http.StatusForbidden, "Operasi tulis ke PostgreSQL tidak diizinkan di environment ini.")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
