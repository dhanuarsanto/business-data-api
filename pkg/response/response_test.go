package response

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestErrorMasking(t *testing.T) {
	run := func() string {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		Error(rec, req, http.StatusInternalServerError, "detail internal rahasia")
		return rec.Body.String()
	}

	Init("development")
	if !strings.Contains(run(), "detail internal rahasia") {
		t.Fatal("mode development harus menampilkan detail error")
	}

	Init("production")
	if strings.Contains(run(), "detail internal rahasia") {
		t.Fatal("mode production tidak boleh membocorkan detail error")
	}
}

func TestCreatedStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	SuccessCreated(rec, req, map[string]string{"message": "ok"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create harus 201, dapat %d", rec.Code)
	}
}
