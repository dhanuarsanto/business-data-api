package http

import (
	"errors"
	"net/http"

	"go.internal/business-data-api/pkg/database"
	"go.internal/business-data-api/pkg/response"
)

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, database.ErrTenantNotFound) {
		response.Error(w, r, http.StatusNotFound, "Tenant tidak ditemukan")
		return
	}
	response.Error(w, r, http.StatusInternalServerError, err.Error())
}