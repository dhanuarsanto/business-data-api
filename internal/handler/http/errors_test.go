package http

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.internal/business-data-api/pkg/database"
	"go.internal/business-data-api/pkg/response"
)

func TestWriteErrorMapping(t *testing.T) {
	response.Init("test")

	run := func(err error) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		writeError(rec, req, err)
		return rec.Code
	}

	if code := run(errors.New("biasa")); code != http.StatusInternalServerError {
		t.Fatalf("error biasa harus 500, dapat %d", code)
	}
	if code := run(database.ErrTenantNotFound); code != http.StatusNotFound {
		t.Fatalf("tenant tak ditemukan harus 404, dapat %d", code)
	}
}