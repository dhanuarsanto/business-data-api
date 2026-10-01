package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	api_middleware "go.internal/business-data-api/internal/middleware"
	"go.internal/business-data-api/internal/usecase"
	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/response"
)

type stubInboxRepo struct {
	items    []dto.InboxItem
	getErr   error
	writeErr error
	inserted domain.Inbox
	updated  dto.UpdateInboxRequest
	filter   domain.InboxFilter
	tenant   string
}

func (s *stubInboxRepo) Get(_ context.Context, tenant string, f domain.InboxFilter) ([]dto.InboxItem, error) {
	s.filter, s.tenant = f, tenant
	return s.items, s.getErr
}
func (s *stubInboxRepo) Insert(_ context.Context, _ string, data domain.Inbox) error {
	s.inserted = data
	return s.writeErr
}
func (s *stubInboxRepo) Update(_ context.Context, _ string, _ int64, req dto.UpdateInboxRequest) error {
	s.updated = req
	return s.writeErr
}

type stubOutboxRepo struct {
	items    []dto.OutboxItem
	getErr   error
	writeErr error
	inserted domain.Outbox
	updated  dto.UpdateOutboxRequest
	filter   domain.OutboxFilter
}

func (s *stubOutboxRepo) Get(_ context.Context, _ string, f domain.OutboxFilter) ([]dto.OutboxItem, error) {
	s.filter = f
	return s.items, s.getErr
}
func (s *stubOutboxRepo) Insert(_ context.Context, _ string, data domain.Outbox) error {
	s.inserted = data
	return s.writeErr
}
func (s *stubOutboxRepo) Update(_ context.Context, _ string, _ int64, req dto.UpdateOutboxRequest) error {
	s.updated = req
	return s.writeErr
}

type stubResellerRepo struct {
	items  []dto.ResellerDropdown
	err    error
	tenant string
}

func (s *stubResellerRepo) ListForDropdown(_ context.Context, tenant string) ([]dto.ResellerDropdown, error) {
	s.tenant = tenant
	return s.items, s.err
}

func reqWithParams(method, target, body string, params map[string]string, headers map[string]string) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, strings.NewReader(body))
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	if len(params) > 0 {
		rctx := chi.NewRouteContext()
		for k, v := range params {
			rctx.URLParams.Add(k, v)
		}
		r = r.WithContext(withRouteCtx(r, rctx))
	}
	return r
}

func withRouteCtx(r *http.Request, rctx *chi.Context) context.Context {
	return context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
}

func newTestConfig() *config.Config {
	return &config.Config{
		AppEnv:                   "test",
		PostgresWriteEnabled:     true,
		JWTTokenDuration:         1 * time.Hour,
		RateLimitLoginRate:       1,
		RateLimitLoginCapacity:   5,
		RateLimitCleanupInterval: time.Minute,
	}
}

func decodeJSON(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var envelope struct {
		Status  string         `json:"status"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("body bukan JSON: %q", rec.Body.String())
	}
	if envelope.Data == nil {
		return map[string]any{}
	}
	return envelope.Data
}

func TestGetInboxMemetakanSemuaParameter(t *testing.T) {
	response.Init("test")

	repo := &stubInboxRepo{items: []dto.InboxItem{{Kode: 55}, {Kode: 77}}}
	h := NewInboxHandler(usecase.NewInboxUsecase(repo, repo))

	target := "/?limit=100&startDate=2026-01-01&endDate=2026-01-31" +
		"&terminal=5&reseller=R1&pengirim=Budi&tipe=S&status=2&pesan=%20halo%20" +
		"&requestFromReseller=true&jawabanFromProvider=1"
	rec := httptest.NewRecorder()
	h.GetInbox(rec, reqWithParams(http.MethodGet, target, "", map[string]string{"tenant": "maxtop"}, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("harus 200, dapat %d body=%s", rec.Code, rec.Body.String())
	}
	f := repo.filter
	if repo.tenant != "maxtop" {
		t.Fatalf("tenant harus diteruskan, dapat %q", repo.tenant)
	}
	if f.Limit != 100 {
		t.Fatalf("limit=%d, harus 100", f.Limit)
	}
	if f.StartDate == nil || f.EndDate == nil {
		t.Fatal("tanggal harus diteruskan")
	}
	if f.Terminal == nil || *f.Terminal != 5 || f.Status == nil || *f.Status != 2 {
		t.Fatal("terminal dan status harus diteruskan")
	}
	if f.Reseller == nil || *f.Reseller != "R1" || f.Pengirim == nil || *f.Pengirim != "Budi" || f.Tipe == nil || *f.Tipe != "S" {
		t.Fatal("filter teks harus diteruskan")
	}
	if f.Pesan != "halo" {
		t.Fatalf("pesan harus dipangkas spasi, dapat %q", f.Pesan)
	}
	if f.RequestFromReseller == nil || !*f.RequestFromReseller || f.JawabanFromProvider == nil || !*f.JawabanFromProvider {
		t.Fatal("flag boolean harus diteruskan")
	}

	body := decodeJSON(t, rec)
	for _, k := range []string{"meta", "trace_id"} {
		if _, ada := body[k]; ada {
			t.Fatalf("key %q tak boleh ada di respons: %v", k, body)
		}
	}
	items, _ := body["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("items=%d, harus 2", len(items))
	}
}

func TestGetInboxParameterRusakDiabaikanDanSumberDefault(t *testing.T) {
	response.Init("test")

	repo := &stubInboxRepo{}
	h := NewInboxHandler(usecase.NewInboxUsecase(repo, repo))

	rec := httptest.NewRecorder()
	target := "/?terminal=abc&status=99999&requestFromReseller=false&jawabanFromProvider=ya"
	h.GetInbox(rec, reqWithParams(http.MethodGet, target, "", map[string]string{"tenant": "maxtop"}, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("harus 200, dapat %d", rec.Code)
	}
	f := repo.filter
	if f.Terminal != nil || f.Status != nil {
		t.Fatal("terminal dan status rusak harus diabaikan")
	}
	if f.RequestFromReseller == nil || *f.RequestFromReseller {
		t.Fatal("nilai selain true/1 harus berarti false")
	}
	if f.JawabanFromProvider == nil || *f.JawabanFromProvider {
		t.Fatal("nilai selain true/1 harus berarti false")
	}
	body := decodeJSON(t, rec)
	if _, ada := body["items"]; !ada {
		t.Fatalf("items harus ada, dapat %v", body)
	}
	if f.Limit != domain.DefaultLimit {
		t.Fatalf("limit rusak harus default, dapat %d", f.Limit)
	}
}

func TestGetInboxMenjalankanRepoSesuaiSumber(t *testing.T) {
	response.Init("test")

	pg := &stubInboxRepo{items: []dto.InboxItem{{Kode: 1}}}
	ms := &stubInboxRepo{items: []dto.InboxItem{{Kode: 2}, {Kode: 3}}}
	h := NewInboxHandler(usecase.NewInboxUsecase(pg, ms))

	for _, c := range []struct {
		sumber string
		want   int64
	}{
		{"", 1},
		{domain.SourcePostgres, 1},
		{domain.SourceMSSQL, 3},
	} {
		rec := httptest.NewRecorder()
		headers := map[string]string{}
		if c.sumber != "" {
			headers["X-DB-Source"] = c.sumber
		}
		h.GetInbox(rec, reqWithParams(http.MethodGet, "/", "", map[string]string{"tenant": "t"}, headers))
		if rec.Code != http.StatusOK {
			t.Fatalf("sumber %q harus 200, dapat %d", c.sumber, rec.Code)
		}
		body := decodeJSON(t, rec)
		items, _ := body["items"].([]any)
		if len(items) == 0 || items[len(items)-1].(map[string]any)["kode"] != float64(c.want) {
			t.Fatalf("sumber %q harus berakhir di kode %d, dapat %v", c.sumber, c.want, items)
		}
	}

	rec := httptest.NewRecorder()
	h.GetInbox(rec, reqWithParams(http.MethodGet, "/", "", map[string]string{"tenant": "t"},
		map[string]string{"X-DB-Source": "oracle"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("sumber tak dikenal harus 400, dapat %d", rec.Code)
	}
}

func TestGetInboxErrorRepoDipetakan(t *testing.T) {
	response.Init("test")

	boom := errors.New("koneksi putus")
	repo := &stubInboxRepo{getErr: boom}
	h := NewInboxHandler(usecase.NewInboxUsecase(repo, repo))

	rec := httptest.NewRecorder()
	h.GetInbox(rec, reqWithParams(http.MethodGet, "/", "", map[string]string{"tenant": "t"}, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("error umum harus 500, dapat %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "koneksi putus") {
		t.Fatal("detail error internal tak boleh bocor ke klien")
	}
}

func TestCreateInboxSemuaCabang(t *testing.T) {
	response.Init("test")
	tenant := map[string]string{"tenant": "maxtop"}

	t.Run("json rusak", func(t *testing.T) {
		repo := &stubInboxRepo{}
		h := NewInboxHandler(usecase.NewInboxUsecase(repo, repo))
		rec := httptest.NewRecorder()
		h.CreateInbox(rec, reqWithParams(http.MethodPost, "/", "{rusak", tenant, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("default tipe pengirim dan seluruh field", func(t *testing.T) {
		repo := &stubInboxRepo{}
		h := NewInboxHandler(usecase.NewInboxUsecase(repo, repo))
		rec := httptest.NewRecorder()
		body := `{"pesan":"halo","pengirim":"Budi","penerima":"Siti","kode_terminal":7,"kode_reseller":"R1","kode_transaksi":9,"service_center":"SC1","status":2,"is_jawaban":1,"is_cs":1,"kode_jawaban_cs":11,"hash":"abc"}`
		h.CreateInbox(rec, reqWithParams(http.MethodPost, "/", body, tenant, nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("harus 201, dapat %d body=%s", rec.Code, rec.Body.String())
		}
		got := repo.inserted
		if got.TipePengirim != "S" {
			t.Fatalf("tipe_pengirim default harus S, dapat %q", got.TipePengirim)
		}
		if got.Pesan != "halo" || got.Pengirim != "Budi" || got.Penerima == nil || *got.Penerima != "Siti" {
			t.Fatalf("field tak diteruskan: %+v", got)
		}
		if got.Status != 2 || got.IsJawaban != 1 || got.KodeTerminal == nil || *got.KodeTerminal != 7 {
			t.Fatalf("field numerik tak diteruskan: %+v", got)
		}
		if got.KodeReseller == nil || *got.KodeReseller != "R1" || got.KodeTransaksi == nil || *got.KodeTransaksi != 9 {
			t.Fatalf("field relasi tak diteruskan: %+v", got)
		}
		if got.ServiceCenter == nil || *got.ServiceCenter != "SC1" || got.IsCs == nil || *got.IsCs != 1 {
			t.Fatalf("field tambahan tak diteruskan: %+v", got)
		}
		if got.KodeJawabanCs == nil || *got.KodeJawabanCs != 11 || got.Hash == nil || *got.Hash != "abc" {
			t.Fatalf("field akhir tak diteruskan: %+v", got)
		}
	})

	t.Run("tipe pengirim eksplisit dipakai", func(t *testing.T) {
		repo := &stubInboxRepo{}
		h := NewInboxHandler(usecase.NewInboxUsecase(repo, repo))
		rec := httptest.NewRecorder()
		h.CreateInbox(rec, reqWithParams(http.MethodPost, "/", `{"pesan":"x","tipe_pengirim":"P"}`, tenant, nil))
		if repo.inserted.TipePengirim != "P" {
			t.Fatalf("tipe_pengirim eksplisit harus dipakai, dapat %q", repo.inserted.TipePengirim)
		}
	})

	t.Run("sumber mssql", func(t *testing.T) {
		pg := &stubInboxRepo{}
		ms := &stubInboxRepo{}
		h := NewInboxHandler(usecase.NewInboxUsecase(pg, ms))
		rec := httptest.NewRecorder()
		h.CreateInbox(rec, reqWithParams(http.MethodPost, "/", `{"pesan":"x"}`, tenant,
			map[string]string{"X-DB-Source": domain.SourceMSSQL}))
		if rec.Code != http.StatusCreated {
			t.Fatalf("harus 201, dapat %d", rec.Code)
		}
		if pg.inserted.Pesan != "" || ms.inserted.Pesan != "x" {
			t.Fatal("sumber mssql harus menulis ke repo mssql")
		}
	})

	t.Run("error repo", func(t *testing.T) {
		repo := &stubInboxRepo{writeErr: errors.New("insert gagal")}
		h := NewInboxHandler(usecase.NewInboxUsecase(repo, repo))
		rec := httptest.NewRecorder()
		h.CreateInbox(rec, reqWithParams(http.MethodPost, "/", `{"pesan":"x"}`, tenant, nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("harus 500, dapat %d", rec.Code)
		}
	})
}

func TestUpdateInboxSemuaCabang(t *testing.T) {
	response.Init("test")

	t.Run("kode bukan angka", func(t *testing.T) {
		h := NewInboxHandler(usecase.NewInboxUsecase(&stubInboxRepo{}, &stubInboxRepo{}))
		rec := httptest.NewRecorder()
		h.UpdateInbox(rec, reqWithParams(http.MethodPut, "/", `{}`,
			map[string]string{"tenant": "t", "kode": "abc"}, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("json rusak", func(t *testing.T) {
		h := NewInboxHandler(usecase.NewInboxUsecase(&stubInboxRepo{}, &stubInboxRepo{}))
		rec := httptest.NewRecorder()
		h.UpdateInbox(rec, reqWithParams(http.MethodPut, "/", "{rusak",
			map[string]string{"tenant": "t", "kode": "5"}, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("sukses dan sumber mssql", func(t *testing.T) {
		pg := &stubInboxRepo{}
		ms := &stubInboxRepo{}
		h := NewInboxHandler(usecase.NewInboxUsecase(pg, ms))
		rec := httptest.NewRecorder()
		body := `{"pesan":"baru","status":3,"pengirim":"Andi","tipe_pengirim":"S","penerima":"Ria","kode_terminal":1,"kode_reseller":"R9","kode_transaksi":2,"is_jawaban":1,"service_center":"SC","is_cs":0,"kode_jawaban_cs":3,"hash":"h"}`
		h.UpdateInbox(rec, reqWithParams(http.MethodPut, "/", body,
			map[string]string{"tenant": "t", "kode": "42"},
			map[string]string{"X-DB-Source": domain.SourceMSSQL}))
		if rec.Code != http.StatusOK {
			t.Fatalf("harus 200, dapat %d body=%s", rec.Code, rec.Body.String())
		}
		got := ms.updated
		if got.Pesan == nil || *got.Pesan != "baru" || got.Status == nil || *got.Status != 3 {
			t.Fatalf("payload tak diteruskan: %+v", got)
		}
		if got.Pengirim == nil || *got.Pengirim != "Andi" || got.Penerima == nil || *got.Penerima != "Ria" {
			t.Fatalf("field teks tak diteruskan: %+v", got)
		}
		if got.KodeTerminal == nil || *got.KodeTerminal != 1 || got.KodeReseller == nil || *got.KodeReseller != "R9" {
			t.Fatalf("field relasi tak diteruskan: %+v", got)
		}
		if got.KodeTransaksi == nil || *got.KodeTransaksi != 2 || got.IsJawaban == nil || *got.IsJawaban != 1 {
			t.Fatalf("field numerik tak diteruskan: %+v", got)
		}
		if got.ServiceCenter == nil || *got.ServiceCenter != "SC" || got.IsCs == nil || *got.IsCs != 0 {
			t.Fatalf("field tambahan tak diteruskan: %+v", got)
		}
		if got.KodeJawabanCs == nil || *got.KodeJawabanCs != 3 || got.Hash == nil || *got.Hash != "h" {
			t.Fatalf("field akhir tak diteruskan: %+v", got)
		}
		if pg.updated.Pesan != nil {
			t.Fatal("repo postgres tak boleh tersentuh")
		}
	})

	t.Run("error repo", func(t *testing.T) {
		repo := &stubInboxRepo{writeErr: errors.New("update gagal")}
		h := NewInboxHandler(usecase.NewInboxUsecase(repo, repo))
		rec := httptest.NewRecorder()
		h.UpdateInbox(rec, reqWithParams(http.MethodPut, "/", `{}`,
			map[string]string{"tenant": "t", "kode": "5"}, nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("harus 500, dapat %d", rec.Code)
		}
	})
}

func TestGetOutboxSemuaCabang(t *testing.T) {
	response.Init("test")

	repo := &stubOutboxRepo{items: []dto.OutboxItem{{Kode: 5}}}
	h := NewOutboxHandler(usecase.NewOutboxUsecase(repo, repo))

	target := "/?reseller=R1&penerima=Siti&tipe=P&status=1&pesan=%20halo%20" +
		"&replyToReseller=true&perintahProvider=1&limit=50"
	rec := httptest.NewRecorder()
	h.GetOutbox(rec, reqWithParams(http.MethodGet, target, "", map[string]string{"tenant": "maxtop"}, nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("harus 200, dapat %d", rec.Code)
	}
	f := repo.filter
	if f.Reseller == nil || *f.Reseller != "R1" || f.Penerima == nil || *f.Penerima != "Siti" || f.Tipe == nil || *f.Tipe != "P" {
		t.Fatalf("filter teks tak diteruskan: %+v", f)
	}
	if f.Status == nil || *f.Status != 1 || f.Pesan != "halo" {
		t.Fatalf("status/pesan tak diteruskan: %+v", f)
	}
	if f.ReplyToReseller == nil || !*f.ReplyToReseller || f.PerintahProvider == nil || !*f.PerintahProvider {
		t.Fatal("flag boolean tak diteruskan")
	}
	if f.Limit != 50 {
		t.Fatalf("limit=%d, harus 50", f.Limit)
	}

	body := decodeJSON(t, rec)
	for _, k := range []string{"meta", "trace_id"} {
		if _, ada := body[k]; ada {
			t.Fatalf("key %q tak boleh ada di respons: %v", k, body)
		}
	}
}

func TestGetOutboxParameterRusakDanError(t *testing.T) {
	response.Init("test")

	repo := &stubOutboxRepo{}
	h := NewOutboxHandler(usecase.NewOutboxUsecase(repo, repo))
	rec := httptest.NewRecorder()
	h.GetOutbox(rec, reqWithParams(http.MethodGet,
		"/?status=abc&replyToReseller=0&perintahProvider=nope", "", map[string]string{"tenant": "t"}, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("harus 200, dapat %d", rec.Code)
	}
	if repo.filter.Status != nil {
		t.Fatal("status rusak harus diabaikan")
	}
	if repo.filter.ReplyToReseller == nil || *repo.filter.ReplyToReseller {
		t.Fatal("replyToReseller selain true/1 harus false")
	}
	if repo.filter.PerintahProvider == nil || *repo.filter.PerintahProvider {
		t.Fatal("perintahProvider selain true/1 harus false")
	}

	gagal := &stubOutboxRepo{getErr: errors.New("boom")}
	h2 := NewOutboxHandler(usecase.NewOutboxUsecase(gagal, gagal))
	rec2 := httptest.NewRecorder()
	h2.GetOutbox(rec2, reqWithParams(http.MethodGet, "/", "", map[string]string{"tenant": "t"}, nil))
	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("harus 500, dapat %d", rec2.Code)
	}
}

func TestCreateOutboxSemuaCabang(t *testing.T) {
	response.Init("test")
	tenant := map[string]string{"tenant": "maxtop"}

	t.Run("json rusak", func(t *testing.T) {
		repo := &stubOutboxRepo{}
		h := NewOutboxHandler(usecase.NewOutboxUsecase(repo, repo))
		rec := httptest.NewRecorder()
		h.CreateOutbox(rec, reqWithParams(http.MethodPost, "/", "{rusak", tenant, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("seluruh field diteruskan", func(t *testing.T) {
		repo := &stubOutboxRepo{}
		h := NewOutboxHandler(usecase.NewOutboxUsecase(repo, repo))
		body := `{"penerima":"Siti","tipe_penerima":"P","pesan":"halo","status":2,"bebas_biaya":1,` +
			`"kode_inbox":10,"kode_transaksi":11,"kode_reseller":"R1","is_perintah":1,"kode_modul":3,` +
			`"prioritas":4,"modul_proses":"MP","pengirim":"Budi","kode_terminal":5,"ctr_kirim":6}`
		rec := httptest.NewRecorder()
		h.CreateOutbox(rec, reqWithParams(http.MethodPost, "/", body, tenant, nil))
		if rec.Code != http.StatusCreated {
			t.Fatalf("harus 201, dapat %d body=%s", rec.Code, rec.Body.String())
		}
		got := repo.inserted
		if got.Penerima != "Siti" || got.TipePenerima != "P" || got.Pesan != "halo" || got.Status != 2 || got.BebasBiaya != 1 {
			t.Fatalf("field utama tak diteruskan: %+v", got)
		}
		if got.KodeInbox == nil || *got.KodeInbox != 10 || got.KodeTransaksi == nil || *got.KodeTransaksi != 11 {
			t.Fatalf("kode tak diteruskan: %+v", got)
		}
		if got.KodeReseller == nil || *got.KodeReseller != "R1" || got.IsPerintah == nil || *got.IsPerintah != 1 {
			t.Fatalf("penanda tak diteruskan: %+v", got)
		}
		if got.KodeModul == nil || *got.KodeModul != 3 || got.Prioritas == nil || *got.Prioritas != 4 {
			t.Fatalf("modul/prioritas tak diteruskan: %+v", got)
		}
		if got.ModulProses == nil || *got.ModulProses != "MP" || got.Pengirim == nil || *got.Pengirim != "Budi" {
			t.Fatalf("teks tak diteruskan: %+v", got)
		}
		if got.KodeTerminal == nil || *got.KodeTerminal != 5 || got.CtrKirim == nil || *got.CtrKirim != 6 {
			t.Fatalf("field akhir tak diteruskan: %+v", got)
		}
	})

	t.Run("error repo dan sumber salah", func(t *testing.T) {
		repo := &stubOutboxRepo{writeErr: errors.New("gagal")}
		h := NewOutboxHandler(usecase.NewOutboxUsecase(repo, repo))
		rec := httptest.NewRecorder()
		h.CreateOutbox(rec, reqWithParams(http.MethodPost, "/", `{"pesan":"x"}`, tenant, nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("harus 500, dapat %d", rec.Code)
		}

		rec2 := httptest.NewRecorder()
		h.CreateOutbox(rec2, reqWithParams(http.MethodPost, "/", `{"pesan":"x"}`, tenant,
			map[string]string{"X-DB-Source": "oracle"}))
		if rec2.Code != http.StatusBadRequest {
			t.Fatalf("sumber tak dikenal harus 400, dapat %d", rec2.Code)
		}
	})
}

func TestUpdateOutboxSemuaCabang(t *testing.T) {
	response.Init("test")

	t.Run("kode bukan angka", func(t *testing.T) {
		h := NewOutboxHandler(usecase.NewOutboxUsecase(&stubOutboxRepo{}, &stubOutboxRepo{}))
		rec := httptest.NewRecorder()
		h.UpdateOutbox(rec, reqWithParams(http.MethodPut, "/", `{}`,
			map[string]string{"tenant": "t", "kode": "xyz"}, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("json rusak", func(t *testing.T) {
		h := NewOutboxHandler(usecase.NewOutboxUsecase(&stubOutboxRepo{}, &stubOutboxRepo{}))
		rec := httptest.NewRecorder()
		h.UpdateOutbox(rec, reqWithParams(http.MethodPut, "/", "{rusak",
			map[string]string{"tenant": "t", "kode": "5"}, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("seluruh field diteruskan", func(t *testing.T) {
		repo := &stubOutboxRepo{}
		h := NewOutboxHandler(usecase.NewOutboxUsecase(repo, repo))
		body := `{"penerima":"Siti","tipe_penerima":"P","pesan":"baru","status":3,"bebas_biaya":1,` +
			`"kode_inbox":10,"kode_transaksi":11,"kode_reseller":"R1","is_perintah":1,"kode_modul":3,` +
			`"prioritas":4,"modul_proses":"MP","pengirim":"Budi","kode_terminal":5,"ctr_kirim":6}`
		rec := httptest.NewRecorder()
		h.UpdateOutbox(rec, reqWithParams(http.MethodPut, "/", body,
			map[string]string{"tenant": "t", "kode": "42"}, nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("harus 200, dapat %d body=%s", rec.Code, rec.Body.String())
		}
		got := repo.updated
		if got.Penerima == nil || *got.Penerima != "Siti" || got.TipePenerima == nil || *got.TipePenerima != "P" {
			t.Fatalf("field utama tak diteruskan: %+v", got)
		}
		if got.Pesan == nil || *got.Pesan != "baru" || got.Status == nil || *got.Status != 3 || got.BebasBiaya == nil || *got.BebasBiaya != 1 {
			t.Fatalf("field numerik tak diteruskan: %+v", got)
		}
		if got.KodeInbox == nil || *got.KodeInbox != 10 || got.KodeTransaksi == nil || *got.KodeTransaksi != 11 {
			t.Fatalf("kode tak diteruskan: %+v", got)
		}
		if got.KodeReseller == nil || *got.KodeReseller != "R1" || got.IsPerintah == nil || *got.IsPerintah != 1 {
			t.Fatalf("penanda tak diteruskan: %+v", got)
		}
		if got.KodeModul == nil || *got.KodeModul != 3 || got.Prioritas == nil || *got.Prioritas != 4 {
			t.Fatalf("modul/prioritas tak diteruskan: %+v", got)
		}
		if got.ModulProses == nil || *got.ModulProses != "MP" || got.Pengirim == nil || *got.Pengirim != "Budi" {
			t.Fatalf("teks tak diteruskan: %+v", got)
		}
		if got.KodeTerminal == nil || *got.KodeTerminal != 5 || got.CtrKirim == nil || *got.CtrKirim != 6 {
			t.Fatalf("field akhir tak diteruskan: %+v", got)
		}
	})

	t.Run("error repo", func(t *testing.T) {
		repo := &stubOutboxRepo{writeErr: errors.New("gagal")}
		h := NewOutboxHandler(usecase.NewOutboxUsecase(repo, repo))
		rec := httptest.NewRecorder()
		h.UpdateOutbox(rec, reqWithParams(http.MethodPut, "/", `{}`,
			map[string]string{"tenant": "t", "kode": "5"}, nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("harus 500, dapat %d", rec.Code)
		}
	})
}

func TestListResellerForDropdownSemuaCabang(t *testing.T) {
	response.Init("test")

	repo := &stubResellerRepo{items: []dto.ResellerDropdown{{Kode: "R1", Nama: "Reseller Satu"}}}
	h := NewMasterHandler(usecase.NewMasterUsecase(repo, repo))

	rec := httptest.NewRecorder()
	h.ListResellerForDropdown(rec, reqWithParams(http.MethodGet, "/", "", map[string]string{"tenant": "maxtop"}, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("harus 200, dapat %d", rec.Code)
	}
	if repo.tenant != "maxtop" {
		t.Fatalf("tenant harus diteruskan, dapat %q", repo.tenant)
	}
	body := decodeJSON(t, rec)
	items, _ := body["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("dropdown harus berisi 1 item, dapat %v", body["items"])
	}

	gagal := &stubResellerRepo{err: errors.New("boom")}
	h2 := NewMasterHandler(usecase.NewMasterUsecase(gagal, gagal))
	rec2 := httptest.NewRecorder()
	h2.ListResellerForDropdown(rec2, reqWithParams(http.MethodGet, "/", "", map[string]string{"tenant": "t"}, nil))
	if rec2.Code != http.StatusInternalServerError {
		t.Fatalf("harus 500, dapat %d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	h2.ListResellerForDropdown(rec3, reqWithParams(http.MethodGet, "/", "", map[string]string{"tenant": "t"},
		map[string]string{"X-DB-Source": "oracle"}))
	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("sumber tak dikenal harus 400, dapat %d", rec3.Code)
	}
}

type stubUserRepo struct {
	user     domain.User
	getErr   error
	writeErr error
	inserted domain.User
	updatedP *string
	updatedR *string
	tenant   string
}

func (s *stubUserRepo) GetByUsername(_ context.Context, tenant, username string) (domain.User, error) {
	s.tenant = tenant
	if s.getErr != nil {
		return domain.User{}, s.getErr
	}
	if s.user.Username != username {
		return domain.User{}, errors.New("tidak ditemukan")
	}
	return s.user, nil
}
func (s *stubUserRepo) Insert(_ context.Context, tenant string, data domain.User) error {
	s.tenant = tenant
	s.inserted = data
	return s.writeErr
}
func (s *stubUserRepo) Update(_ context.Context, tenant, username string, password, rules *string) error {
	s.tenant = tenant
	s.updatedP, s.updatedR = password, rules
	return s.writeErr
}

func TestLoginSemuaCabang(t *testing.T) {
	response.Init("test")
	jwt.InitJWT("rahasia-uji-128bit", 3600000000000, "business-data-api")

	resolver := api_middleware.NewTrustedProxyResolver(nil)
	matrix := map[string]bool{"*": true, "sa": false}

	t.Run("json rusak", func(t *testing.T) {
		repo := &stubUserRepo{}
		h := NewAuthHandler(usecase.NewAuthUsecase(repo, repo), matrix, resolver, false)
		rec := httptest.NewRecorder()
		h.Login(rec, reqWithParams(http.MethodPost, "/", "{rusak", map[string]string{"tenant": "t"}, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("kredensial salah", func(t *testing.T) {
		repo := &stubUserRepo{user: domain.User{Username: "budi", Password: "rahasia", Rules: "sa"}}
		h := NewAuthHandler(usecase.NewAuthUsecase(repo, repo), matrix, resolver, false)
		rec := httptest.NewRecorder()
		h.Login(rec, reqWithParams(http.MethodPost, "/", `{"username":"budi","password":"salah"}`,
			map[string]string{"tenant": "t"}, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("harus 401, dapat %d", rec.Code)
		}
	})

	t.Run("peran wajib lokal ditolak dari luar", func(t *testing.T) {
		repo := &stubUserRepo{user: domain.User{UserID: 1, Username: "budi", Password: "rahasia", Rules: "rahasia"}}
		h := NewAuthHandler(usecase.NewAuthUsecase(repo, repo), map[string]bool{"rahasia": true}, resolver, false)
		rec := httptest.NewRecorder()
		req := reqWithParams(http.MethodPost, "/", `{"username":"budi","password":"rahasia"}`,
			map[string]string{"tenant": "t"}, nil)
		req.RemoteAddr = "203.0.113.5:1234"
		h.Login(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("harus 403, dapat %d", rec.Code)
		}
	})

	t.Run("sukses dari jaringan lokal", func(t *testing.T) {
		repo := &stubUserRepo{user: domain.User{UserID: 1, Username: "budi", Password: "rahasia", Rules: "sa"}}
		h := NewAuthHandler(usecase.NewAuthUsecase(repo, repo), map[string]bool{"sa": true}, resolver, true)
		h.sessionMaxAge = 3600
		rec := httptest.NewRecorder()
		req := reqWithParams(http.MethodPost, "/", `{"username":"budi","password":"rahasia"}`,
			map[string]string{"tenant": "maxtop"}, nil)
		req.RemoteAddr = "10.0.0.5:1234"
		h.Login(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("harus 200, dapat %d body=%s", rec.Code, rec.Body.String())
		}
		if repo.tenant != "maxtop" {
			t.Fatalf("tenant harus diteruskan, dapat %q", repo.tenant)
		}
		cookies := rec.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Name != "access_token" || !cookies[0].HttpOnly || !cookies[0].Secure {
			t.Fatalf("cookie sesi tidak benar: %+v", cookies)
		}
		body := decodeJSON(t, rec)
		if body["token"] == nil || body["token"] == "" || body["username"] != "budi" || body["rules"] != "sa" {
			t.Fatalf("body login salah: %v", body)
		}
	})
}

func TestCreateUserSemuaCabang(t *testing.T) {
	response.Init("test")
	tenant := map[string]string{"tenant": "t"}

	t.Run("json rusak", func(t *testing.T) {
		h := NewAuthHandler(usecase.NewAuthUsecase(&stubUserRepo{}, &stubUserRepo{}), nil, api_middleware.NewTrustedProxyResolver(nil), false)
		rec := httptest.NewRecorder()
		h.CreateUser(rec, reqWithParams(http.MethodPost, "/", "{rusak", tenant, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("field kosong ditolak", func(t *testing.T) {
		h := NewAuthHandler(usecase.NewAuthUsecase(&stubUserRepo{}, &stubUserRepo{}), nil, api_middleware.NewTrustedProxyResolver(nil), false)
		rec := httptest.NewRecorder()
		h.CreateUser(rec, reqWithParams(http.MethodPost, "/", `{"username":"","password":"x","rules":"sa"}`, tenant, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("sukses ke sumber mssql", func(t *testing.T) {
		pg := &stubUserRepo{}
		ms := &stubUserRepo{}
		h := NewAuthHandler(usecase.NewAuthUsecase(pg, ms), nil, api_middleware.NewTrustedProxyResolver(nil), false)
		rec := httptest.NewRecorder()
		h.CreateUser(rec, reqWithParams(http.MethodPost, "/", `{"username":"budi","password":"rahasia","rules":"sa"}`,
			tenant, map[string]string{"X-DB-Source": domain.SourceMSSQL}))
		if rec.Code != http.StatusCreated {
			t.Fatalf("harus 201, dapat %d body=%s", rec.Code, rec.Body.String())
		}
		if ms.inserted.Username != "budi" || ms.inserted.Rules != "sa" {
			t.Fatalf("user tak diteruskan: %+v", ms.inserted)
		}
		if pg.inserted.Username != "" {
			t.Fatal("repo postgres tak boleh tersentuh")
		}
	})

	t.Run("error repo dan sumber salah", func(t *testing.T) {
		repo := &stubUserRepo{writeErr: errors.New("gagal")}
		h := NewAuthHandler(usecase.NewAuthUsecase(repo, repo), nil, api_middleware.NewTrustedProxyResolver(nil), false)
		rec := httptest.NewRecorder()
		h.CreateUser(rec, reqWithParams(http.MethodPost, "/", `{"username":"budi","password":"x","rules":"sa"}`, tenant, nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("harus 500, dapat %d", rec.Code)
		}

		rec2 := httptest.NewRecorder()
		h.CreateUser(rec2, reqWithParams(http.MethodPost, "/", `{"username":"budi","password":"x","rules":"sa"}`,
			tenant, map[string]string{"X-DB-Source": "oracle"}))
		if rec2.Code != http.StatusBadRequest {
			t.Fatalf("sumber tak dikenal harus 400, dapat %d", rec2.Code)
		}
	})
}

func TestUpdateUserSemuaCabang(t *testing.T) {
	response.Init("test")
	params := map[string]string{"tenant": "t", "username": "budi"}

	t.Run("json rusak", func(t *testing.T) {
		h := NewAuthHandler(usecase.NewAuthUsecase(&stubUserRepo{}, &stubUserRepo{}), nil, api_middleware.NewTrustedProxyResolver(nil), false)
		rec := httptest.NewRecorder()
		h.UpdateUser(rec, reqWithParams(http.MethodPut, "/", "{rusak", params, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("password kosong ditolak", func(t *testing.T) {
		h := NewAuthHandler(usecase.NewAuthUsecase(&stubUserRepo{}, &stubUserRepo{}), nil, api_middleware.NewTrustedProxyResolver(nil), false)
		rec := httptest.NewRecorder()
		h.UpdateUser(rec, reqWithParams(http.MethodPut, "/", `{"password":"","rules":"op"}`, params, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("harus 400, dapat %d", rec.Code)
		}
	})

	t.Run("sukses dan sumber mssql", func(t *testing.T) {
		pg := &stubUserRepo{}
		ms := &stubUserRepo{}
		h := NewAuthHandler(usecase.NewAuthUsecase(pg, ms), nil, api_middleware.NewTrustedProxyResolver(nil), false)
		rec := httptest.NewRecorder()
		h.UpdateUser(rec, reqWithParams(http.MethodPut, "/", `{"password":"rahasia2","rules":"op"}`,
			params, map[string]string{"X-DB-Source": domain.SourceMSSQL}))
		if rec.Code != http.StatusOK {
			t.Fatalf("harus 200, dapat %d body=%s", rec.Code, rec.Body.String())
		}
		if ms.updatedP == nil || *ms.updatedP != "rahasia2" || ms.updatedR == nil || *ms.updatedR != "op" {
			t.Fatalf("payload tak diteruskan: %v %v", ms.updatedP, ms.updatedR)
		}
		if pg.updatedP != nil {
			t.Fatal("repo postgres tak boleh tersentuh")
		}
	})

	t.Run("error repo dan sumber salah", func(t *testing.T) {
		repo := &stubUserRepo{writeErr: errors.New("gagal")}
		h := NewAuthHandler(usecase.NewAuthUsecase(repo, repo), nil, api_middleware.NewTrustedProxyResolver(nil), false)
		rec := httptest.NewRecorder()
		h.UpdateUser(rec, reqWithParams(http.MethodPut, "/", `{"rules":"op"}`, params, nil))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("harus 500, dapat %d", rec.Code)
		}

		rec2 := httptest.NewRecorder()
		h.UpdateUser(rec2, reqWithParams(http.MethodPut, "/", `{"rules":"op"}`, params,
			map[string]string{"X-DB-Source": "oracle"}))
		if rec2.Code != http.StatusBadRequest {
			t.Fatalf("sumber tak dikenal harus 400, dapat %d", rec2.Code)
		}
	})
}

func TestRateLimitersSebelumDanSesudahRegister(t *testing.T) {
	response.Init("test")

	repo := &stubUserRepo{}
	h := NewAuthHandler(usecase.NewAuthUsecase(repo, repo), nil, api_middleware.NewTrustedProxyResolver(nil), false)
	if got := h.RateLimiters(); got != nil {
		t.Fatalf("sebelum RegisterRoutes harus nil, dapat %v", got)
	}

	router := chi.NewRouter()
	cfg := newTestConfig()
	h.RegisterRoutes(router, router, cfg, map[string][]string{"ManageUsers": {"sa"}})
	defer func() {
		for _, l := range h.RateLimiters() {
			l.Stop()
		}
	}()

	limiters := h.RateLimiters()
	if len(limiters) != 1 || limiters[0] != h.authLimiter {
		t.Fatalf("harus mengembalikan satu limiter milik handler, dapat %v", limiters)
	}
	if h.sessionMaxAge != int(cfg.JWTTokenDuration.Seconds()) {
		t.Fatalf("sessionMaxAge harus mengikuti durasi token, dapat %d", h.sessionMaxAge)
	}
}

func TestRegisterRoutesInboxOutboxMasterTerpasang(t *testing.T) {
	response.Init("test")

	inboxRepo := &stubInboxRepo{}
	outboxRepo := &stubOutboxRepo{}
	resellerRepo := &stubResellerRepo{}

	router := chi.NewRouter()
	cfg := newTestConfig()
	inbox := NewInboxHandler(usecase.NewInboxUsecase(inboxRepo, inboxRepo))
	outbox := NewOutboxHandler(usecase.NewOutboxUsecase(outboxRepo, outboxRepo))
	master := NewMasterHandler(usecase.NewMasterUsecase(resellerRepo, resellerRepo))

	roleMatrix := map[string][]string{
		"ReadInbox": {"sa"}, "WriteInbox": {"sa"},
		"ReadOutbox": {"sa"}, "WriteOutbox": {"sa"},
		"ReadResellerDropdown": {"sa"},
	}
	inbox.RegisterRoutes(router, router, cfg, roleMatrix)
	outbox.RegisterRoutes(router, router, cfg, roleMatrix)
	master.RegisterRoutes(router, router, cfg, roleMatrix)

	routes := map[string]struct{}{}
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		routes[method+" "+route] = struct{}{}
		return nil
	}); err != nil {
		t.Fatalf("walk router gagal: %v", err)
	}
	for _, want := range []struct{ method, path string }{
		{"GET", "/api/v1/{tenant}/inbox/"},
		{"POST", "/api/v1/{tenant}/inbox/"},
		{"PUT", "/api/v1/{tenant}/inbox/{kode}"},
		{"GET", "/api/v1/{tenant}/outbox/"},
		{"POST", "/api/v1/{tenant}/outbox/"},
		{"PUT", "/api/v1/{tenant}/outbox/{kode}"},
		{"GET", "/api/v1/{tenant}/master/reseller-dropdown"},
	} {
		if _, ada := routes[want.method+" "+want.path]; !ada {
			t.Fatalf("rute %s %s tak terdaftar", want.method, want.path)
		}
	}
}
