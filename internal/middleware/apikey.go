package middleware

import (
	"encoding/json"
	"net/http"
	"os"

	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"
)

func APIKeyValidator() func(http.Handler) http.Handler {
	keys := make(map[string]string)
	fileData, err := os.ReadFile("api_keys.json")
	if err == nil {
		json.Unmarshal(fileData, &keys)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tCtx := logger.GetTraceContext(r.Context())
			key := r.Header.Get("X-API-KEY")

			if key == "" {
				response.Error(w, r, http.StatusUnauthorized, "Akses ditolak: Header X-API-KEY kosong")
				return
			}

			if err != nil {
				response.Error(w, r, http.StatusInternalServerError, "Internal Error: Gagal membaca file api_keys.json")
				return
			}

			developerName, exists := keys[key]
			if !exists {
				response.Error(w, r, http.StatusUnauthorized, "Akses ditolak: API Key tidak terdaftar")
				return
			}

			tCtx.Developer = developerName
			next.ServeHTTP(w, r)
		})
	}
}
