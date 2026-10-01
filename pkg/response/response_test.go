package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"go.internal/business-data-api/pkg/logger"
)

func withInit(t *testing.T, env string) {
	t.Helper()
	sebelum := hideInternalDetails
	t.Cleanup(func() { hideInternalDetails = sebelum })
	Init(env)
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) JSONResponse {
	t.Helper()
	var out JSONResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body bukan JSON: %q", rec.Body.String())
	}
	return out
}

func TestInitMenentukanPenyembunyianDetail(t *testing.T) {
	withInit(t, "production")
	if !hideInternalDetails {
		t.Fatal("lingkungan selain development wajib menyembunyikan detail")
	}
	withInit(t, "development")
	if hideInternalDetails {
		t.Fatal("development boleh menampilkan detail")
	}
}

func TestSuccessMengirimStatusDanData(t *testing.T) {
	withInit(t, "test")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/maxtop/inbox", nil)
	req = req.WithContext(logger.SetTraceContext(req.Context(), &logger.TraceContext{TraceID: "tr-1"}))

	Success(rec, req, map[string]any{"items": []int{1, 2}})

	if rec.Code != http.StatusOK {
		t.Fatalf("harus 200, dapat %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content type salah: %q", ct)
	}
	body := decodeBody(t, rec)
	if body.Status != "sukses" || body.Message != "" {
		t.Fatalf("envelope tak sesuai: %+v", body)
	}
	data, ok := body.Data.(map[string]any)
	if !ok || data["items"] == nil {
		t.Fatalf("data harus diteruskan, dapat %#v", body.Data)
	}
}

func TestSuccessCreatedMemakaiStatusDibuat(t *testing.T) {
	withInit(t, "test")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/maxtop/inbox", nil)

	SuccessCreated(rec, req, map[string]any{"kode": 9})

	if rec.Code != http.StatusCreated {
		t.Fatalf("harus 201, dapat %d", rec.Code)
	}
	if decodeBody(t, rec).Status != "sukses" {
		t.Fatal("status body harus sukses")
	}
}

func TestSuccessTanpaDataTetapValid(t *testing.T) {
	withInit(t, "test")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	Success(rec, req, nil)

	body := decodeBody(t, rec)
	if body.Status != "sukses" || body.Data != nil {
		t.Fatalf("data kosong tak boleh diikutsertakan: %s", rec.Body.String())
	}
}

func TestErrorMenyembunyikanDetailSaatProduksi(t *testing.T) {
	withInit(t, "production")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	cases := []struct {
		status int
		want   string
	}{
		{http.StatusInternalServerError, "Terjadi kesalahan internal pada server"},
		{http.StatusBadGateway, "Terjadi kesalahan internal pada server"},
		{http.StatusUnauthorized, "Akses ditolak"},
		{http.StatusForbidden, "Akses ditolak"},
		{http.StatusNotFound, "Resource tidak ditemukan"},
		{http.StatusBadRequest, "Permintaan tidak valid"},
		{http.StatusTooManyRequests, "Batas request terlampaui. Silakan coba sesaat lagi."},
		{http.StatusMethodNotAllowed, "Method HTTP tidak diizinkan pada endpoint ini"},
		{http.StatusRequestEntityTooLarge, "Ukuran body permintaan melebihi batas"},
		{http.StatusConflict, "Permintaan tidak dapat diproses"},
		{http.StatusUnprocessableEntity, "Permintaan tidak dapat diproses"},
		{http.StatusNotImplemented, "Terjadi kesalahan internal pada server"},
		{http.StatusOK, "Permintaan tidak dapat diproses"},
	}

	var mu sync.Mutex
	for _, c := range cases {
		mu.Lock()
		rec := httptest.NewRecorder()
		Error(rec, req, c.status, "rahasia internal yang tak boleh bocor")
		mu.Unlock()

		if rec.Code != c.status {
			t.Fatalf("status %d harus diteruskan, dapat %d", c.status, rec.Code)
		}
		body := decodeBody(t, rec)
		if body.Status != "gagal" {
			t.Fatalf("status body harus gagal, dapat %q", body.Status)
		}
		if body.Message != c.want {
			t.Fatalf("status %d: pesan %q harus %q", c.status, body.Message, c.want)
		}
	}
}

func TestErrorMenampilkanDetailSaatDevelopment(t *testing.T) {
	withInit(t, "development")
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	for _, status := range []int{http.StatusInternalServerError, http.StatusUnauthorized,
		http.StatusForbidden, http.StatusNotFound, http.StatusBadRequest, http.StatusConflict} {
		rec := httptest.NewRecorder()
		Error(rec, req, status, "detail asli untuk pengembang")

		body := decodeBody(t, rec)
		if body.Message != "detail asli untuk pengembang" {
			t.Fatalf("status %d: development harus menampilkan detail, dapat %q", status, body.Message)
		}
	}
}

func TestErrorTanpaTraceContextMenggunakanDefault(t *testing.T) {
	withInit(t, "development")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	Error(rec, req, http.StatusConflict, "konflik")

	if rec.Code != http.StatusConflict {
		t.Fatalf("harus 409, dapat %d", rec.Code)
	}
	if decodeBody(t, rec).Message != "konflik" {
		t.Fatalf("pesan tak sesuai: %q", rec.Body.String())
	}
}

func TestCursorPaginationMetaMenyembunyikanKursorNol(t *testing.T) {
	ada := CursorPaginationMeta{HasNextPage: true, HasPrevPage: false, NextCursor: 42}
	if ada.NextCursor != 42 || !ada.HasNextPage || ada.HasPrevPage {
		t.Fatalf("nilai meta tak sesuai: %+v", ada)
	}

	tanpa := CursorPaginationMeta{}
	hasil, err := json.Marshal(tanpa)
	if err != nil {
		t.Fatalf("marshal gagal: %v", err)
	}
	if string(hasil) != `{"has_next_page":false,"has_prev_page":false}` {
		t.Fatalf("kursor nol harus dihilangkan, dapat %s", hasil)
	}
}
