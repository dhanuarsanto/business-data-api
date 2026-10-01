package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.internal/business-data-api/internal/domain"
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
	if code := run(domain.ErrSourceNotValid); code != http.StatusBadRequest {
		t.Fatalf("sumber tidak valid harus 400, dapat %d", code)
	}
}

func TestWriteErrorTidakBocorkanPesanInternal(t *testing.T) {
	response.Init("test")

	rahasia := "pq: relation \"rahasia_internal\" does not exist"
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	writeError(rec, req, errors.New(rahasia))

	body := rec.Body.String()
	for _, bocor := range []string{"rahasia_internal", "pq:", "relation"} {
		if strings.Contains(body, bocor) {
			t.Fatalf("pesan internal bocor ke client (%q): %s", bocor, body)
		}
	}

	var parsed map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("respons harus JSON valid: %v", err)
	}
	if msg, _ := parsed["message"].(string); msg == "" {
		t.Fatalf("client harus tetap menerima pesan umum: %s", rec.Body.String())
	}
	if parsed["status"] != "gagal" {
		t.Fatalf("status harus gagal, dapat %s", rec.Body.String())
	}
}
