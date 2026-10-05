package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/internal/domain"
	api_middleware "go.internal/business-data-api/internal/middleware"
	"go.internal/business-data-api/internal/usecase"
	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/response"
)

type modulUji struct {
	terdaftar bool
	limiter   []*api_middleware.RateLimiter
}

func (m *modulUji) RegisterRoutes(_, _ chi.Router, _ *config.Config, _ map[string][]string) {
	m.terdaftar = true
}

func (m *modulUji) RateLimiters() []*api_middleware.RateLimiter { return m.limiter }

const kunciUji = "kunci-uji-router"

func setupRouter(t *testing.T) (*config.Config, *api_middleware.TrustedProxyResolver) {
	t.Helper()
	cfg := newTestConfig()
	cfg.MaxBodyBytes = 1 << 20
	cfg.APIKeysPath = filepath.Join(t.TempDir(), "api_keys.json")
	if err := os.WriteFile(cfg.APIKeysPath, []byte(`{"`+kunciUji+`":"dev-uji"}`), 0o600); err != nil {
		t.Fatalf("tulis api_keys.json gagal: %v", err)
	}
	cfg.AllowedOrigins = "http://a.test"
	cfg.CORSMaxAge = 60
	cfg.RateLimitGlobalRate = 1000
	cfg.RateLimitGlobalCapacity = 1000
	cfg.GlobalLocalOnly = true
	return cfg, api_middleware.NewTrustedProxyResolver(nil)
}

func requestLokal(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-API-KEY", kunciUji)
	return req
}

func TestLoginMenolakSaatJWTBelumDiinisialisasi(t *testing.T) {
	response.Init("test")
	t.Cleanup(func() { jwt.InitJWT("rahasia-uji-128bit", time.Hour, "business-data-api") })
	jwt.InitJWT("", time.Hour, "business-data-api")

	repo := &stubUserRepo{user: domain.User{UserID: 1, Username: "budi", Password: "rahasia", Rules: "sa"}}
	h := NewAuthHandler(usecase.NewAuthUsecase(repo, repo), map[string]bool{"sa": true},
		api_middleware.NewTrustedProxyResolver(nil), false)

	rec := httptest.NewRecorder()
	req := reqWithParams(http.MethodPost, "/", `{"username":"budi","password":"rahasia"}`,
		map[string]string{"tenant": "maxtop"}, nil)
	req.RemoteAddr = "10.0.0.5:1234"
	h.Login(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("tanpa rahasia JWT harus 500, dapat %d body=%s", rec.Code, rec.Body.String())
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("tak ada cookie yang boleh terbit saat token gagal dibuat")
	}
}

func TestSwaggerYAMLPathMemakaiBerkasLokalJikaAda(t *testing.T) {
	asal, err := os.Getwd()
	if err != nil {
		t.Fatalf("ambil cwd gagal: %v", err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("buat docs gagal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "swagger.yaml"), []byte("swagger: \"2.0\"\n"), 0o600); err != nil {
		t.Fatalf("tulis swagger gagal: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir gagal: %v", err)
	}
	t.Cleanup(func() { os.Chdir(asal) })

	got := swaggerYAMLPath()

	if !filepath.IsAbs(got) {
		t.Fatalf("jalur harus absolut, dapat %q", got)
	}
	if filepath.Base(filepath.Dir(got)) != "docs" || filepath.Base(got) != "swagger.yaml" {
		t.Fatalf("jalur tak sesuai: %q", got)
	}
}

func TestSwaggerYAMLPathKembaliKeKandidatAwalSaatBerkasTidakAda(t *testing.T) {
	asal, err := os.Getwd()
	if err != nil {
		t.Fatalf("ambil cwd gagal: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir gagal: %v", err)
	}
	t.Cleanup(func() { os.Chdir(asal) })

	exe, err := os.Executable()
	if err != nil {
		if got := swaggerYAMLPath(); got != "docs/swagger.yaml" {
			t.Fatalf("tanpa executable harus jatuh ke kandidat awal, dapat %q", got)
		}
		return
	}
	calon := []string{
		filepath.Join(filepath.Dir(exe), "docs", "swagger.yaml"),
		filepath.Join(filepath.Dir(exe), "..", "docs", "swagger.yaml"),
	}
	for _, c := range calon {
		if _, err := os.Stat(c); err == nil {
			t.Skipf("berkas swagger tersedia di %q, skenario tak berlaku", c)
		}
	}
	if got := swaggerYAMLPath(); got != "docs/swagger.yaml" {
		t.Fatalf("tanpa berkas harus jatuh ke kandidat awal, dapat %q", got)
	}
}

func TestSetupRoutesMendaftarkanModulDanLimiterTambahan(t *testing.T) {
	cfg, resolver := setupRouter(t)
	limiterTambahan := api_middleware.NewRateLimiter(1000, 1000, resolver, time.Minute)
	modul := &modulUji{limiter: []*api_middleware.RateLimiter{limiterTambahan}}

	r, limiters, keyManager := SetupRoutes(cfg, resolver, map[string]bool{"127.0.0.1": true}, nil, modul)

	if !modul.terdaftar {
		t.Fatal("RegisterRoutes modul harus dipanggil")
	}
	if len(limiters) != 2 {
		t.Fatalf("harus ada limiter global + limiter modul, dapat %d", len(limiters))
	}
	if limiters[1] != limiterTambahan {
		t.Fatal("limiter modul harus ikut dikembalikan")
	}
	if keyManager == nil || r == nil {
		t.Fatal("router dan key manager tak boleh nil")
	}
}

func TestSetupRoutesTanpaModulTetapJalan(t *testing.T) {
	cfg, resolver := setupRouter(t)

	r, limiters, keyManager := SetupRoutes(cfg, resolver, map[string]bool{"127.0.0.1": true}, nil)

	if len(limiters) != 1 {
		t.Fatalf("tanpa modul harus ada satu limiter global, dapat %d", len(limiters))
	}
	if keyManager == nil || r == nil {
		t.Fatal("router dan key manager tak boleh nil")
	}
}

func TestSetupRoutesMemakaiNamaBerkasKunciBawaanSaatKosong(t *testing.T) {
	cfg, resolver := setupRouter(t)
	cfg.APIKeysPath = ""

	asal, err := os.Getwd()
	if err != nil {
		t.Fatalf("ambil cwd gagal: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir gagal: %v", err)
	}
	t.Cleanup(func() { os.Chdir(asal) })

	if _, _, _ = SetupRoutes(cfg, resolver, map[string]bool{"127.0.0.1": true}, nil); false {
		t.Fatal("tidak mungkin")
	}

	if _, err := os.Stat(filepath.Join(dir, "api_keys.json")); err == nil {
		t.Fatal("berkas api_keys.json tak boleh dibuat di direktori kerja")
	}
	km, _, _ := SetupRoutes(cfg, resolver, map[string]bool{"127.0.0.1": true}, nil)
	if km == nil {
		t.Fatal("key manager tak boleh nil walau nama berkasnya kosong")
	}
}

func TestSetupRoutesMelayaniHealthDanDokumen(t *testing.T) {
	cfg, resolver := setupRouter(t)
	r, _, _ := SetupRoutes(cfg, resolver, map[string]bool{"127.0.0.1": true}, nil)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, requestLokal(http.MethodGet, "/health"))
	if rec.Code != http.StatusOK {
		t.Fatalf("/health harus dilayani, dapat %d: %s", rec.Code, rec.Body.String())
	}

	recUI := httptest.NewRecorder()
	r.ServeHTTP(recUI, requestLokal(http.MethodGet, "/swagger/index.html"))
	if recUI.Code != http.StatusOK {
		t.Fatalf("UI swagger harus dilayani, dapat %d", recUI.Code)
	}

	recTanpaKunci := httptest.NewRecorder()
	tanpaKunci := httptest.NewRequest(http.MethodGet, "/health", nil)
	tanpaKunci.RemoteAddr = "127.0.0.1:5555"
	r.ServeHTTP(recTanpaKunci, tanpaKunci)
	if recTanpaKunci.Code != http.StatusUnauthorized {
		t.Fatalf("tanpa X-API-KEY harus 401, dapat %d", recTanpaKunci.Code)
	}

	asal, err := os.Getwd()
	if err != nil {
		t.Fatalf("ambil cwd gagal: %v", err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatalf("buat docs gagal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "swagger.yaml"), []byte("swagger: \"2.0\"\n"), 0o600); err != nil {
		t.Fatalf("tulis swagger gagal: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir gagal: %v", err)
	}
	t.Cleanup(func() { os.Chdir(asal) })

	recYAML := httptest.NewRecorder()
	r.ServeHTTP(recYAML, requestLokal(http.MethodGet, "/docs/swagger.yaml"))
	if recYAML.Code != http.StatusOK {
		t.Fatalf("berkas swagger harus dilayani, dapat %d", recYAML.Code)
	}
	if recYAML.Body.String() != "swagger: \"2.0\"\n" {
		t.Fatalf("isi swagger tak sesuai: %q", recYAML.Body.String())
	}
}

func TestSetupRoutesMenutupJalurTakDikenalDenganAman(t *testing.T) {
	cfg, resolver := setupRouter(t)
	r, _, _ := SetupRoutes(cfg, resolver, map[string]bool{"127.0.0.1": true}, nil)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, requestLokal(http.MethodGet, "/tidak-ada"))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("jalur tak dikenal harus 404, dapat %d", rec.Code)
	}

	recMethod := httptest.NewRecorder()
	r.ServeHTTP(recMethod, requestLokal(http.MethodDelete, "/health"))
	if recMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method tak diizinkan harus 405, dapat %d", recMethod.Code)
	}
}

func TestSetupRoutesDitolakSaatBadanTerlaluBesar(t *testing.T) {
	cfg, resolver := setupRouter(t)
	cfg.MaxBodyBytes = 8
	r, _, _ := SetupRoutes(cfg, resolver, map[string]bool{"127.0.0.1": true}, nil)

	req := httptest.NewRequest(http.MethodPost, "/health", strings.NewReader(strings.Repeat("x", 64)))
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-API-KEY", kunciUji)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("badan berlebihan harus 413, dapat %d", rec.Code)
	}
}

func TestSetupRoutesMemakaiOriginYangDiizinkan(t *testing.T) {
	cfg, resolver := setupRouter(t)
	cfg.AllowedOrigins = "http://diizinkan.test"
	r, _, _ := SetupRoutes(cfg, resolver, map[string]bool{"127.0.0.1": true}, nil)

	req := requestLokal(http.MethodGet, "/health")
	req.Header.Set("Origin", "http://dilarang.test")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "http://dilarang.test" {
		t.Fatal("origin tak diizinkan tak boleh dipantulkan")
	}

	reqIzin := requestLokal(http.MethodGet, "/health")
	reqIzin.Header.Set("Origin", "http://diizinkan.test")
	recIzin := httptest.NewRecorder()
	r.ServeHTTP(recIzin, reqIzin)
	if got := recIzin.Header().Get("Access-Control-Allow-Origin"); got != "http://diizinkan.test" {
		t.Fatalf("origin diizinkan harus dipantulkan, dapat %q", got)
	}
}

func TestSetupRoutesMeloloskanPrefixTanpaResolverKosong(t *testing.T) {
	cfg, resolver := setupRouter(t)
	cfg.GlobalLocalOnly = false
	cfg.AllowDirectClients = true
	r, _, _ := SetupRoutes(cfg, resolver, nil, nil)

	req := requestLokal(http.MethodGet, "/health")
	req.RemoteAddr = "203.0.113.9:5555"
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("klien langsung yang diizinkan harus bisa,/health dapat %d", rec.Code)
	}

	cfg2, resolver2 := setupRouter(t)
	r2, _, _ := SetupRoutes(cfg2, resolver2, map[string]bool{"127.0.0.1": true}, nil)
	reqLokal := requestLokal(http.MethodGet, "/health")
	recLokal := httptest.NewRecorder()
	r2.ServeHTTP(recLokal, reqLokal)
	if recLokal.Code != http.StatusOK {
		t.Fatalf("klien lokal harus selalu bisa, dapat %d", recLokal.Code)
	}
}
