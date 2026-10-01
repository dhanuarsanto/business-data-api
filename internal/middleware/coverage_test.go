package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"
)

func withClaims(r *http.Request, claims map[string]any) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), claimsKey, claims))
}

func withTenant(r *http.Request, tenant string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("tenant", tenant)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestKeyManagerMiddlewareMembalasStatusTepat(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")

	path := filepath.Join(t.TempDir(), "api_keys.json")
	if err := os.WriteFile(path, []byte(`{"kunci-uji":"Budi"}`), 0o600); err != nil {
		t.Fatalf("tidak bisa menulis berkas uji: %v", err)
	}

	km := NewKeyManager(path)
	defer km.Stop()

	run := func(key string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		if key != "" {
			req.Header.Set("X-API-KEY", key)
		}
		req = req.WithContext(logger.SetTraceContext(req.Context(), &logger.TraceContext{}))
		var masuk bool
		km.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			masuk = true
			if got := logger.GetTraceContext(r.Context()).Developer; got != "Budi" {
				t.Errorf("developer harus tersimpan di trace context, dapat %q", got)
			}
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)
		if masuk && rec.Code != http.StatusOK {
			t.Fatalf("handler dipanggil tapi status %d", rec.Code)
		}
		return rec.Code
	}

	if code := run(""); code != http.StatusUnauthorized {
		t.Fatalf("kunci kosong harus 401, dapat %d", code)
	}
	if code := run("kunci-ngawur"); code != http.StatusUnauthorized {
		t.Fatalf("kunci tak terdaftar harus 401, dapat %d", code)
	}
	if code := run("kunci-uji"); code != http.StatusOK {
		t.Fatalf("kunci terdaftar harus 200, dapat %d", code)
	}
}

func TestKeyManagerReloadGagalMempertahankanDaftarLama(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")

	path := filepath.Join(t.TempDir(), "api_keys.json")
	if err := os.WriteFile(path, []byte(`{"kunci-uji":"Budi"}`), 0o600); err != nil {
		t.Fatalf("tidak bisa menulis berkas uji: %v", err)
	}

	km := NewKeyManager(path)
	defer km.Stop()

	if _, ok := km.Lookup("kunci-uji"); !ok {
		t.Fatal("kunci awal harus terbaca")
	}
	if err := km.reload(); err != nil {
		t.Fatalf("reload pertama harus sukses: %v", err)
	}
	if err := os.WriteFile(path, []byte(`bukan json`), 0o600); err != nil {
		t.Fatalf("tidak bisa merusak berkas uji: %v", err)
	}
	if err := km.reload(); err == nil {
		t.Fatal("reload berkas rusak harus gagal")
	}
	if _, ok := km.Lookup("kunci-uji"); !ok {
		t.Fatal("reload gagal tak boleh menghapus daftar lama")
	}
}

func TestNewKeyManagerBerkasHilangAkanKosong(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")

	km := NewKeyManager(filepath.Join(t.TempDir(), "tidak-ada.json"))
	defer km.Stop()

	if _, ok := km.Lookup("apa-saja"); ok {
		t.Fatal("berkas hilang berarti tidak ada kunci")
	}
}

func TestRequireTokenMenolakSemuaVarianHeaderRusak(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")
	jwt.InitJWT("rahasia-uji-128bit", time.Hour, "business-data-api")

	token, err := jwt.GenerateToken(1, "budi", "sa", "maxtop")
	if err != nil {
		t.Fatalf("token uji gagal dibuat: %v", err)
	}

	cases := []struct {
		nama   string
		header string
	}{
		{"header kosong", ""},
		{"tanpa skema", token},
		{"skema salah", "Basic " + token},
		{"terlalu banyak bagian", "Bearer a b"},
		{"token rusak", "Bearer bukan.token.jwt"},
	}

	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := withTenant(httptest.NewRequest(http.MethodGet, "/", nil), "maxtop")
			if c.header != "" {
				req.Header.Set("Authorization", c.header)
			}
			RequireToken()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("harus 401, dapat %d", rec.Code)
			}
		})
	}
}

func TestRequireTokenMenolakTenantTidakCocokDanKosong(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")
	jwt.InitJWT("rahasia-uji-128bit", time.Hour, "business-data-api")

	token, err := jwt.GenerateToken(7, "budi", "sa", "maxtop")
	if err != nil {
		t.Fatalf("token uji gagal dibuat: %v", err)
	}

	run := func(tenant string) int {
		rec := httptest.NewRecorder()
		req := withTenant(httptest.NewRequest(http.MethodGet, "/", nil), tenant)
		req.Header.Set("Authorization", "Bearer "+token)
		var claimsCtx bool
		RequireToken()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claimsCtx = r.Context().Value(claimsKey) != nil
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)
		if claimsCtx && rec.Code != http.StatusOK {
			t.Fatal("claims baru harus tersimpan kalau lolos")
		}
		return rec.Code
	}

	if code := run("tenant-lain"); code != http.StatusForbidden {
		t.Fatalf("tenant tak cocok harus 403, dapat %d", code)
	}
	if code := run(""); code != http.StatusForbidden {
		t.Fatalf("tenant kosong harus 403, dapat %d", code)
	}
}

func TestRequireRoleTolakTanpaClaimsDanRoleAneh(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")
	jwt.InitJWT("rahasia-uji-128bit", time.Hour, "business-data-api")

	cases := []struct {
		nama      string
		claims    map[string]any
		diizinkan []string
		want      int
	}{
		{"tanpa claims", nil, []string{"sa"}, http.StatusForbidden},
		{"claims tanpa rules", map[string]any{"user_id": 1.0}, []string{"sa"}, http.StatusForbidden},
		{"role tidak diizinkan", map[string]any{"rules": "tamu"}, []string{"sa"}, http.StatusForbidden},
		{"role diizinkan", map[string]any{"rules": "sa"}, []string{"sa", "op"}, http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if c.claims != nil {
				req = withClaims(req, c.claims)
			}
			rec := httptest.NewRecorder()
			RequireRole(c.diizinkan...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("harus %d, dapat %d", c.want, rec.Code)
			}
		})
	}
}

func TestNetworkRoleGuardSemuaCabang(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")

	resolver := NewTrustedProxyResolver(nil)
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cases := []struct {
		nama       string
		globalOnly bool
		roleMatrix map[string]bool
		remote     string
		claims     map[string]any
		want       int
	}{
		{"global only tolak remote publik", true, nil, "203.0.113.5:1234", nil, http.StatusForbidden},
		{"global only terima remote lokal", true, nil, "10.0.0.5:1234", nil, http.StatusOK},
		{"tanpa claims lolos saat global only mati", false, map[string]bool{"rahasia": true}, "203.0.113.5:1234", nil, http.StatusOK},
		{"role publik boleh dari luar", false, map[string]bool{"rahasia": true, "publik": false}, "203.0.113.5:1234", map[string]any{"rules": "publik"}, http.StatusOK},
		{"role lokal ditolak dari luar", false, map[string]bool{"rahasia": true, "publik": false}, "203.0.113.5:1234", map[string]any{"rules": "rahasia"}, http.StatusForbidden},
		{"role lokal diterima dari dalam", false, map[string]bool{"rahasia": true, "publik": false}, "10.0.0.5:1234", map[string]any{"rules": "rahasia"}, http.StatusOK},
		{"role tak dikenal ikut bintang lokal", false, map[string]bool{"*": true}, "203.0.113.5:1234", map[string]any{"rules": "asing"}, http.StatusForbidden},
		{"role tak dikenal tanpa bintang lolos", false, map[string]bool{"rahasia": true}, "203.0.113.5:1234", map[string]any{"rules": "asing"}, http.StatusOK},
		{"remote tidak valid dianggap bukan lokal", false, map[string]bool{"*": true}, "bukan-alamat", map[string]any{"rules": "asing"}, http.StatusForbidden},
	}

	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = c.remote
			if c.claims != nil {
				req = withClaims(req, c.claims)
			}
			rec := httptest.NewRecorder()
			NetworkRoleGuard(c.globalOnly, c.roleMatrix, resolver)(ok).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("harus %d, dapat %d", c.want, rec.Code)
			}
		})
	}
}

func TestIsRoleAllowedFromOutside(t *testing.T) {
	lokal := map[string]bool{"rahasia": true, "publik": false}

	if !IsRoleAllowedFromOutside("publik", lokal) {
		t.Fatal("peran publik boleh dari luar")
	}
	if IsRoleAllowedFromOutside("rahasia", lokal) {
		t.Fatal("peran lokal tak boleh dari luar")
	}
	if IsRoleAllowedFromOutside("asing", map[string]bool{"*": true}) {
		t.Fatal("bintang lokal berarti peran asing tetap dilarang dari luar")
	}
	if !IsRoleAllowedFromOutside("asing", lokal) {
		t.Fatal("tanpa entri dan tanpa bintang berarti bebas dari luar")
	}
	if !IsRoleAllowedFromOutside("apa-saja", map[string]bool{}) {
		t.Fatal("matriks kosong berarti tanpa peran lokal")
	}
}

func TestPostgresWriteGuardSemuaMetodeDanSumber(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")

	cases := []struct {
		nama    string
		method  string
		sumber  string
		enabled bool
		want    int
	}{
		{"tulis postgres saat dimatikan", http.MethodPost, "postgres", false, http.StatusForbidden},
		{"ubah postgres saat dimatikan", http.MethodPut, "postgres", false, http.StatusForbidden},
		{"hapus postgres saat dimatikan", http.MethodDelete, "postgres", false, http.StatusForbidden},
		{"sumber kosong dianggap postgres", http.MethodPost, "", false, http.StatusForbidden},
		{"tulis postgres saat diizinkan", http.MethodPost, "postgres", true, http.StatusOK},
		{"tulis mssql tidak terpengaruh", http.MethodPost, "mssql", false, http.StatusOK},
		{"baca postgres tidak terpengaruh", http.MethodGet, "postgres", false, http.StatusOK},
		{"baca mssql tidak terpengaruh", http.MethodGet, "mssql", false, http.StatusOK},
		{"sumber huruf besar tidak dianggap postgres", http.MethodPost, "POSTGRES", false, http.StatusOK},
	}

	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			req := httptest.NewRequest(c.method, "/", nil)
			if c.sumber != "" {
				req.Header.Set("X-DB-Source", c.sumber)
			}
			rec := httptest.NewRecorder()
			PostgresWriteGuard(c.enabled)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("harus %d, dapat %d", c.want, rec.Code)
			}
		})
	}
}

func TestRateLimiterMengisiUlangDanMembersihkan(t *testing.T) {
	rl := NewRateLimiter(0, 2, nil, time.Minute)
	defer rl.Stop()

	if !rl.allow("ip-uji") {
		t.Fatal("permintaan pertama harus lolos")
	}
	if !rl.allow("ip-uji") {
		t.Fatal("permintaan kedua harus lolos")
	}
	if rl.allow("ip-uji") {
		t.Fatal("permintaan ketiga harus ditolak saat token habis")
	}
	if !rl.allow("ip-lain") {
		t.Fatal("IP berbeda punya token sendiri")
	}

	rl.visitors["ip-uji"].lastRefill = time.Now().Add(-time.Hour)
	rl.visitors["ip-lain"].lastRefill = time.Now()
	rl.cleanup(time.Minute)
	if _, ada := rl.visitors["ip-uji"]; ada {
		t.Fatal("pengunjung lama harus dibersihkan")
	}
	if _, ada := rl.visitors["ip-lain"]; !ada {
		t.Fatal("pengunjung baru harus tetap ada")
	}
}

func TestRateLimiterIsiUlangSesuaiLaju(t *testing.T) {
	rl := NewRateLimiter(1000, 1, nil, time.Minute)
	defer rl.Stop()

	if !rl.allow("ip") {
		t.Fatal("permintaan pertama harus lolos")
	}
	if rl.allow("ip") {
		t.Fatal("token kedua harus terlampaui")
	}
	rl.visitors["ip"].lastRefill = time.Now().Add(-10 * time.Millisecond)
	if !rl.allow("ip") {
		t.Fatal("token harus terisi ulang setelah jeda")
	}
}

func TestRateLimiterMiddlewareTolakSaatHabis(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")

	rl := NewRateLimiter(0, 1, nil, time.Minute)
	defer rl.Stop()

	panggil := func() int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rl.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)
		return rec.Code
	}

	if panggil() != http.StatusOK {
		t.Fatal("permintaan pertama harus lolos")
	}
	if panggil() != http.StatusTooManyRequests {
		t.Fatal("permintaan kedua harus 429")
	}
}

func TestRateLimiterStopAmanDipanggilBerkaliKali(t *testing.T) {
	rl := NewRateLimiter(1, 1, nil, time.Minute)
	rl.Stop()
	rl.Stop()
}
