package middleware

import (
	"net/http"

	"go.internal/business-data-api/pkg/response"
)

func RequestBodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				response.Error(w, r, http.StatusRequestEntityTooLarge, "Ukuran body permintaan melebihi batas")
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}