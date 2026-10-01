package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.internal/business-data-api/pkg/response"
)

func TestClientAddrMenanganiSeluruhVarianHeader(t *testing.T) {
	proxy := NewTrustedProxyResolver([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	tanpaPrefix := NewTrustedProxyResolver(nil)

	cases := []struct {
		nama     string
		resolver *TrustedProxyResolver
		remote   string
		header   map[string]string
		want     string
	}{
		{"remote kosong", proxy, "", nil, "invalid IP"},
		{"remote bukan alamat", proxy, "bukan-alamat", nil, "invalid IP"},
		{"proxy tak dipercaya memakai remote", proxy, "203.0.113.7:9000", nil, "203.0.113.7"},
		{"proxy dipercaya tanpa header", proxy, "10.0.0.3:9000", nil, "10.0.0.3"},
		{"resolver tanpa prefix memakai remote", tanpaPrefix, "10.0.0.3:9000",
			map[string]string{"X-Forwarded-For": "198.51.100.9"}, "10.0.0.3"},
		{"x-forwarded-for dipakai", proxy, "10.0.0.3:9000",
			map[string]string{"X-Forwarded-For": "198.51.100.9, 203.0.113.7"}, "203.0.113.7"},
		{"rantai x-forwarded-for seluruhnya proxy", proxy, "10.0.0.3:9000",
			map[string]string{"X-Forwarded-For": "10.0.0.4, 10.0.0.5"}, "10.0.0.3"},
		{"x-forwarded-for rusak dihentikan", proxy, "10.0.0.3:9000",
			map[string]string{"X-Forwarded-For": "203.0.113.7, rusak"}, "10.0.0.3"},
		{"x-real-ip dipakai", proxy, "10.0.0.3:9000",
			map[string]string{"X-Real-IP": "198.51.100.1"}, "198.51.100.1"},
		{"x-real-ip rusak diabaikan", proxy, "10.0.0.3:9000",
			map[string]string{"X-Real-IP": "bukan-alamat"}, "10.0.0.3"},
		{"ipv4-mapped ipv6 dinormalkan", proxy, "[::ffff:10.0.0.3]:9000", nil, "10.0.0.3"},
		{"bukan alamat dengan port", proxy, "0x7f.1:9000", nil, "invalid IP"},
	}

	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = c.remote
			for k, v := range c.header {
				req.Header.Set(k, v)
			}
			if got := c.resolver.clientAddr(req).String(); got != c.want {
				t.Fatalf("clientAddr harus %q, dapat %q", c.want, got)
			}
		})
	}
}

func TestNewRateLimiterMemakaiIntervalBawaanSaatTidakPositif(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Second} {
		rl := NewRateLimiter(1, 1, nil, d)
		if rl == nil {
			t.Fatal("limiter tak boleh nil")
		}
		rl.Stop()
	}
}

func TestCleanupMembuangHanyaPengunjungKedaluwarsa(t *testing.T) {
	rl := NewRateLimiter(1, 1, nil, time.Minute)
	defer rl.Stop()

	rl.mu.Lock()
	rl.visitors["lama"] = &clientVisitor{tokens: 1, lastRefill: time.Now().Add(-time.Hour)}
	rl.visitors["baru"] = &clientVisitor{tokens: 1, lastRefill: time.Now()}
	rl.mu.Unlock()

	rl.cleanup(time.Minute)

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if _, ada := rl.visitors["lama"]; ada {
		t.Fatal("pengunjung kedaluwarsa harus dibuang")
	}
	if _, ada := rl.visitors["baru"]; !ada {
		t.Fatal("pengunjung aktif harus tetap ada")
	}
}

func TestRunCleanupMenjalankanPembersihanBerkala(t *testing.T) {
	rl := NewRateLimiter(1, 1, nil, 5*time.Millisecond)
	defer rl.Stop()

	rl.mu.Lock()
	rl.visitors["lama"] = &clientVisitor{tokens: 1, lastRefill: time.Now().Add(-time.Hour)}
	rl.mu.Unlock()

	tungguSampai(t, func() bool {
		rl.mu.Lock()
		defer rl.mu.Unlock()
		_, masihLama := rl.visitors["lama"]
		return !masihLama
	}, "runCleanup harus membuang pengunjung kedaluwarsa secara berkala")
}

func TestRunCleanupBerhentiSaatStopDipanggil(t *testing.T) {
	rl := &RateLimiter{
		visitors: make(map[string]*clientVisitor),
		rate:     1,
		capacity: 1,
		done:     make(chan struct{}),
	}
	selesai := make(chan struct{})
	go func() {
		rl.runCleanup(time.Millisecond)
		close(selesai)
	}()

	rl.Stop()
	select {
	case <-selesai:
	case <-time.After(2 * time.Second):
		t.Fatal("runCleanup harus berhenti setelah done ditutup")
	}
	rl.Stop()
}

func TestRunReloadMemuatUlangDanMempertahankanDaftarSaatGagal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "api_keys.json")
	tulis := func(isi string) {
		if err := os.WriteFile(path, []byte(isi), 0o600); err != nil {
			t.Fatalf("tulis gagal: %v", err)
		}
	}
	tulis(`{"kunci-lama":"Dev Lama"}`)

	km := NewKeyManager(path)
	defer km.Stop()
	go km.runReload(2 * time.Millisecond)

	tulis(`bukan json`)
	waktu := time.Now().Add(300 * time.Millisecond)
	for time.Now().Before(waktu) {
		if dev, ok := km.Lookup("kunci-lama"); !ok || dev != "Dev Lama" {
			t.Fatalf("reload gagal harus mempertahankan daftar lama, dapat %q %v", dev, ok)
		}
		time.Sleep(5 * time.Millisecond)
	}

	tulis(`{"kunci-baru":"Dev Baru"}`)
	tungguSampai(t, func() bool {
		_, ok := km.Lookup("kunci-baru")
		return ok
	}, "reload berkala harus membaca daftar baru")
}

func TestRateLimiterMiddlewareMemakaiAlamatKlienDariProxy(t *testing.T) {
	response.Init("test")
	resolver := NewTrustedProxyResolver([]netip.Prefix{netip.MustParsePrefix("10.0.0.0/8")})
	rl := NewRateLimiter(0, 1, resolver, time.Minute)
	defer rl.Stop()

	var dipanggil int
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dipanggil++
		w.WriteHeader(http.StatusOK)
	})

	for i := range 3 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.3:9000"
		req.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i+1))
		rec := httptest.NewRecorder()
		rl.Middleware()(next).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("klien %d punya jatah sendiri, harus 200, dapat %d", i, rec.Code)
		}
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	if len(rl.visitors) != 3 || dipanggil != 3 {
		t.Fatalf("tiap klien harus punya wadahnya sendiri: visitors=%d dipanggil=%d", len(rl.visitors), dipanggil)
	}
}

func tungguSampai(t *testing.T, syarat func() bool, pesanGagal string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if syarat() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal(pesanGagal)
		}
		time.Sleep(2 * time.Millisecond)
	}
}
