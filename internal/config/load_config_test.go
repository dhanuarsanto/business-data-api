package config

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func chdirSementara(t *testing.T, dir string) {
	t.Helper()
	asal, err := os.Getwd()
	if err != nil {
		t.Fatalf("ambil cwd gagal: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir gagal: %v", err)
	}
	t.Cleanup(func() { os.Chdir(asal) })
}

func bersihkanLingkungan(t *testing.T) {
	t.Helper()
	asli := map[string]string{}
	var nama []string
	for _, e := range os.Environ() {
		if i := strings.IndexByte(e, '='); i >= 0 {
			nama = append(nama, e[:i])
			asli[e[:i]] = e[i+1:]
		}
	}
	t.Cleanup(func() {
		for _, n := range nama {
			os.Setenv(n, asli[n])
		}
	})
	for _, n := range nama {
		os.Unsetenv(n)
	}
}

func TestLoadConfigMembacaBerkasEnvDanBawaan(t *testing.T) {
	dir := t.TempDir()
	bersihkanLingkungan(t)
	chdirSementara(t, dir)

	isi := "APP_ENV=staging\n" +
		"PORT=9091\n" +
		"GLOBAL_LOCAL_ONLY=false\n" +
		"POSTGRES_WRITE_ENABLED=true\n" +
		"COOKIE_SECURE=false\n" +
		"ALLOWED_ORIGINS=http://a.test,http://b.test\n" +
		"TRUSTED_PROXIES=10.0.0.0/8\n" +
		"MAX_BODY_BYTES=2048\n" +
		"POSTGRES_MAX_CONNS=7\n" +
		"POSTGRES_MAX_CONN_IDLE_TIME=90s\n" +
		"POSTGRES_MAX_CONN_LIFETIME=45m\n" +
		"POSTGRES_HEALTH_CHECK_PERIOD=15s\n" +
		"MSSQL_MAX_OPEN_CONNS=5\n" +
		"MSSQL_MAX_IDLE_CONNS=2\n" +
		"MSSQL_MAX_CONN_LIFETIME=30m\n" +
		"RATE_LIMIT_GLOBAL_RATE=12.5\n" +
		"RATE_LIMIT_GLOBAL_CAPACITY=25\n" +
		"RATE_LIMIT_LOGIN_RATE=0.5\n" +
		"RATE_LIMIT_LOGIN_CAPACITY=3\n" +
		"RATE_LIMIT_CLEANUP_INTERVAL=2m\n" +
		"CORS_MAX_AGE=600\n" +
		"JWT_SECRET=rahasia-yang-panjang-sekali-untuk-uji\n" +
		"JWT_ISSUER=issuer-uji\n" +
		"JWT_TOKEN_DURATION=3h\n" +
		"POSTGRES_MAXTOP_URL=postgres://a/b\n" +
		"MSSQL_MAXTOP_URL=sqlserver://a/b\n" +
		"POSTGRES_PANDORA_URL=postgres://c/d\n" +
		"MSSQL_PANDORA_URL=sqlserver://c/d\n" +
		"POSTGRES_TOPLINK_URL=postgres://e/f\n" +
		"MSSQL_TOPLINK_URL=sqlserver://e/f\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(isi), 0o600); err != nil {
		t.Fatalf("tulis .env gagal: %v", err)
	}

	cfg := LoadConfig()

	if cfg.AppEnv != "staging" || cfg.Port != 9091 {
		t.Fatalf("nilai dasar tak terbaca: env=%q port=%d", cfg.AppEnv, cfg.Port)
	}
	if cfg.GlobalLocalOnly || !cfg.PostgresWriteEnabled || cfg.CookieSecure {
		t.Fatalf("bendera bool tak terbaca: %+v", cfg)
	}
	if len(cfg.GetAllowedOrigins()) != 2 {
		t.Fatalf("origin tak terbaca: %v", cfg.GetAllowedOrigins())
	}
	if cfg.MaxBodyBytes != 2048 {
		t.Fatalf("batas badan tak terbaca: %d", cfg.MaxBodyBytes)
	}
	if cfg.PostgresMaxConns != 7 || cfg.PostgresMinConns != 0 {
		t.Fatalf("batas koneksi postgres tak terbaca: max=%d min=%d", cfg.PostgresMaxConns, cfg.PostgresMinConns)
	}
	if cfg.PostgresMaxConnIdleTime != 90*time.Second || cfg.PostgresMaxConnLifetime != 45*time.Minute {
		t.Fatalf("usia koneksi postgres tak terbaca: %+v", cfg)
	}
	if cfg.PostgresHealthCheckPeriod != 15*time.Second {
		t.Fatalf("periode health check tak terbaca: %v", cfg.PostgresHealthCheckPeriod)
	}
	if cfg.MSSQLMaxOpenConns != 5 || cfg.MSSQLMaxIdleConns != 2 || cfg.MSSQLMaxConnLifetime != 30*time.Minute {
		t.Fatalf("opsi mssql tak terbaca: %+v", cfg)
	}
	if cfg.RateLimitGlobalRate != 12.5 || cfg.RateLimitGlobalCapacity != 25 {
		t.Fatalf("batas global tak terbaca: %+v", cfg)
	}
	if cfg.RateLimitLoginRate != 0.5 || cfg.RateLimitLoginCapacity != 3 || cfg.RateLimitCleanupInterval != 2*time.Minute {
		t.Fatalf("batas login tak terbaca: %+v", cfg)
	}
	if cfg.CORSMaxAge != 600 {
		t.Fatalf("umur CORS tak terbaca: %d", cfg.CORSMaxAge)
	}
	if cfg.JWTTokenDuration != 3*time.Hour || cfg.JWTIssuer != "issuer-uji" {
		t.Fatalf("konfigurasi JWT tak terbaca: %v %q", cfg.JWTTokenDuration, cfg.JWTIssuer)
	}
	if cfg.APIKeysPath != "api_keys.json" {
		t.Fatalf("bawaan API_KEYS_PATH berubah: %q", cfg.APIKeysPath)
	}
	if cfg.AllowDirectClients {
		t.Fatal("ALLOW_DIRECT_CLIENTS bawaan harus false")
	}
}

func setWajib(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"POSTGRES_MAXTOP_URL":  "postgres://a/b",
		"MSSQL_MAXTOP_URL":     "sqlserver://a/b",
		"POSTGRES_PANDORA_URL": "postgres://c/d",
		"MSSQL_PANDORA_URL":    "sqlserver://c/d",
		"POSTGRES_TOPLINK_URL": "postgres://e/f",
		"MSSQL_TOPLINK_URL":    "sqlserver://e/f",
		"JWT_SECRET":           "rahasia-yang-panjang-sekali-untuk-uji",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadConfigLanjutSaatBerkasEnvTidakAda(t *testing.T) {
	dir := t.TempDir()
	bersihkanLingkungan(t)
	chdirSementara(t, dir)
	setWajib(t)

	cfg := LoadConfig()

	if cfg.AppEnv != "development" {
		t.Fatalf("bawaan APP_ENV harus development, dapat %q", cfg.AppEnv)
	}
	if !cfg.GlobalLocalOnly || cfg.PostgresWriteEnabled || !cfg.CookieSecure {
		t.Fatalf("bawaan keamanan berubah: %+v", cfg)
	}
	if cfg.Port != 8080 || cfg.MaxBodyBytes != 1048576 {
		t.Fatalf("bawaan port/badan berubah: %d %d", cfg.Port, cfg.MaxBodyBytes)
	}
	if cfg.TrustedProxies != "" || cfg.AllowDirectClients {
		t.Fatalf("bawaan proxy berubah: %q %v", cfg.TrustedProxies, cfg.AllowDirectClients)
	}
}

func TestLoadConfigMelanjutkanSaatBerkasEnvTidakDapatDibaca(t *testing.T) {
	dir := t.TempDir()
	bersihkanLingkungan(t)
	chdirSementara(t, dir)

	// Direktori bernama .env membuat pembacaan gagal tanpa menjadi "tidak ada".
	if err := os.Mkdir(filepath.Join(dir, ".env"), 0o755); err != nil {
		t.Fatalf("buat direktori .env gagal: %v", err)
	}

	setWajib(t)

	cfg := LoadConfig()

	if cfg.AppEnv != "development" {
		t.Fatalf("konfigurasi tak boleh gagal hanya karena .env tak terbaca, dapat %q", cfg.AppEnv)
	}
}

const penandaSubproses = "LOAD_CONFIG_SUBPROSES"

// LoadConfig memanggil os.Exit(1) saat env.Parse gagal, jadi jalurnya diuji
// lewat proses anak: proses anak memanggil LoadConfig tanpa variabel wajib.
func TestLoadConfigKeluarSaatVariabelWajibHilang(t *testing.T) {
	if os.Getenv(penandaSubproses) == "1" {
		dir := t.TempDir()
		bersihkanLingkunganTanpaRestore(t)
		chdirSementara(t, dir)
		LoadConfig()
		t.Fatal("LoadConfig harus keluar sendiri, bukan kembali")
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLoadConfigKeluarSaatVariabelWajibHilang$", "-test.count=1")
	cmd.Env = append(os.Environ(), penandaSubproses+"=1")
	keluaran, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("proses anak harus keluar dengan kode 1, dapat err=%v keluaran=%s", err, keluaran)
	}
	if strings.Contains(string(keluaran), "PASS") {
		t.Fatalf("proses anak seharusnya gagal: %s", keluaran)
	}
	if !strings.Contains(string(keluaran), "POSTGRES_MAXTOP_URL") {
		t.Fatalf("pesan kegagalan harus menyebut variabel wajib, dapat: %s", keluaran)
	}
}

// bersihkanLingkunganTanpaRestore mengosongkan seluruh lingkungan agar
// variabel wajib benar-benar hilang. GOCOVERDIR dipertahankan supaya
// cakupan dari proses anak tetap digabung ke profil utama.
func bersihkanLingkunganTanpaRestore(t *testing.T) {
	t.Helper()
	cakupan := os.Getenv("GOCOVERDIR")
	for _, e := range os.Environ() {
		if i := strings.IndexByte(e, '='); i >= 0 {
			os.Unsetenv(e[:i])
		}
	}
	if cakupan != "" {
		os.Setenv("GOCOVERDIR", cakupan)
	}
}
