package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const penandaProsesAPI = "CMD_API_SUBPROSES"

// main() menutup proses dengan os.Exit(1) di setiap jalur kegagalan, jadi hanya
// bisa diuji lewat proses anak. Anak memakai direktori kerja sementara supaya
// logs/api.log dari logger.SetupLogger tidak pernah menyentuh repo.
func TestMain(m *testing.M) {
	if os.Getenv(penandaProsesAPI) == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

var konfigurasiApp = []string{
	"APP_ENV", "GLOBAL_LOCAL_ONLY", "POSTGRES_WRITE_ENABLED", "PORT", "ALLOWED_ORIGINS",
	"TRUSTED_PROXIES", "ALLOW_DIRECT_CLIENTS", "MAX_BODY_BYTES", "COOKIE_SECURE", "API_KEYS_PATH",
	"POSTGRES_MAXTOP_URL", "MSSQL_MAXTOP_URL", "POSTGRES_PANDORA_URL", "MSSQL_PANDORA_URL",
	"POSTGRES_TOPLINK_URL", "MSSQL_TOPLINK_URL", "POSTGRES_MAX_CONNS", "POSTGRES_MIN_CONNS",
	"POSTGRES_MAX_CONN_IDLE_TIME", "POSTGRES_MAX_CONN_LIFETIME", "POSTGRES_HEALTH_CHECK_PERIOD",
	"MSSQL_MAX_OPEN_CONNS", "MSSQL_MAX_IDLE_CONNS", "MSSQL_MAX_CONN_LIFETIME",
	"RATE_LIMIT_GLOBAL_RATE", "RATE_LIMIT_GLOBAL_CAPACITY", "RATE_LIMIT_LOGIN_RATE",
	"RATE_LIMIT_LOGIN_CAPACITY", "RATE_LIMIT_CLEANUP_INTERVAL", "SERVER_READ_HEADER_TIMEOUT",
	"SERVER_READ_TIMEOUT", "SERVER_WRITE_TIMEOUT", "SERVER_IDLE_TIMEOUT", "SERVER_SHUTDOWN_TIMEOUT",
	"CORS_MAX_AGE", "JWT_SECRET", "JWT_ISSUER", "JWT_TOKEN_DURATION",
}

// lingkunganAnak menyisakan hanya variabel yang dibutuhkan sistem, membuang
// seluruh variabel konfigurasi milik aplikasi supaya hasil uji tidak ikut
// berubah-ubah mengikuti shell developers. GOCOVERDIR tetap ikut agar cakupan
// main() digabung ke profil utama.
func lingkunganAnak(t *testing.T, tambahan map[string]string) []string {
	t.Helper()
	buang := make(map[string]bool, len(konfigurasiApp)+1)
	for _, n := range konfigurasiApp {
		buang[n] = true
	}
	buang[penandaProsesAPI] = true

	var hasil []string
	for _, e := range os.Environ() {
		i := strings.IndexByte(e, '=')
		if i < 0 || buang[e[:i]] {
			continue
		}
		hasil = append(hasil, e)
	}
	for k, v := range tambahan {
		hasil = append(hasil, k+"="+v)
	}
	return append(hasil, penandaProsesAPI+"=1")
}

func jalankanMain(t *testing.T, tambahan map[string]string) (string, int, string) {
	t.Helper()
	dir := t.TempDir()

	ctx, batal := context.WithTimeout(context.Background(), 90*time.Second)
	defer batal()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^$", "-test.count=1")
	cmd.Dir = dir
	cmd.Env = lingkunganAnak(t, tambahan)

	keluaran, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return string(keluaran), 0, dir
	case errors.As(err, &exitErr):
		return string(keluaran), exitErr.ExitCode(), dir
	default:
		t.Fatalf("gagal menjalankan proses anak: %v", err)
		return "", -1, dir
	}
}

// dsnMati menunjuk ke port 1 yang tidak pernah mendengarkan, jadi koneksi pasti
// ditolak tanpa pernah menyentuh server database mana pun.
func konfigurasiValid() map[string]string {
	return map[string]string{
		"APP_ENV":                 "development",
		"POSTGRES_MAXTOP_URL":     "postgres://uji:uji@127.0.0.1:1/uji?sslmode=disable&connect_timeout=2",
		"MSSQL_MAXTOP_URL":        "sqlserver://uji:uji@127.0.0.1:1?database=uji&connection+timeout=2&dial+timeout=2",
		"POSTGRES_PANDORA_URL":    "postgres://uji:uji@127.0.0.1:1/uji?sslmode=disable&connect_timeout=2",
		"MSSQL_PANDORA_URL":       "sqlserver://uji:uji@127.0.0.1:1?database=uji&connection+timeout=2&dial+timeout=2",
		"POSTGRES_TOPLINK_URL":    "postgres://uji:uji@127.0.0.1:1/uji?sslmode=disable&connect_timeout=2",
		"MSSQL_TOPLINK_URL":       "sqlserver://uji:uji@127.0.0.1:1?database=uji&connection+timeout=2&dial+timeout=2",
		"JWT_SECRET":              strings.Repeat("rahasia", 6),
		"JWT_ISSUER":              "business-data-api",
		"SERVER_SHUTDOWN_TIMEOUT": "2s",
	}
}

func TestMainMenolakSecretYangTerlaluPendek(t *testing.T) {
	env := konfigurasiValid()
	env["JWT_SECRET"] = "pendek"

	keluaran, kode, dir := jalankanMain(t, env)

	if kode != 1 {
		t.Fatalf("harus keluar dengan kode 1, dapat %d: %s", kode, keluaran)
	}
	if !strings.Contains(keluaran, "JWT_SECRET terlalu pendek") {
		t.Fatalf("harus menolak secret pendek dengan jelas, dapat: %s", keluaran)
	}
	if strings.Contains(keluaran, "Gagal koneksi Postgres") {
		t.Fatalf("harus berhenti sebelum menyentuh database, dapat: %s", keluaran)
	}
	if _, err := os.Stat(filepath.Join(dir, "logs")); !os.IsNotExist(err) {
		t.Fatalf("logger belum boleh dijalankan, tapi logs/ sudah ada: %v", err)
	}
}

func TestMainBerhentiSaatPostgresMaxtopTidakTerhubung(t *testing.T) {
	keluaran, kode, dir := jalankanMain(t, konfigurasiValid())

	if kode != 1 {
		t.Fatalf("harus keluar dengan kode 1, dapat %d: %s", kode, keluaran)
	}
	if !strings.Contains(keluaran, "Menjalankan API") {
		t.Fatalf("harus mencatat mode sebelum connects, dapat: %s", keluaran)
	}
	if !strings.Contains(keluaran, "Gagal koneksi Postgres Maxtop") {
		t.Fatalf("harus melaporkan kegagalan Maxtop, dapat: %s", keluaran)
	}
	if strings.Contains(keluaran, "Gagal koneksi MSSQL") || strings.Contains(keluaran, "Gagal koneksi Postgres Pandora") {
		t.Fatalf("harus berhenti di kegagalan pertama, dapat: %s", keluaran)
	}
	if _, err := os.Stat(filepath.Join(dir, "logs", "api.log")); err != nil {
		t.Fatalf("logger harus sudah menulis logs/api.log di direktori kerja anak: %v", err)
	}
}

func TestMainGagalSaatKonfigurasiWajibKosong(t *testing.T) {
	env := konfigurasiValid()
	delete(env, "POSTGRES_TOPLINK_URL")
	delete(env, "MSSQL_TOPLINK_URL")
	env["MSSQL_MAXTOP_URL"] = ""

	keluaran, kode, _ := jalankanMain(t, env)

	if kode != 1 {
		t.Fatalf("harus keluar dengan kode 1, dapat %d: %s", kode, keluaran)
	}
	if !strings.Contains(keluaran, "Gagal mem-parsing konfigurasi") {
		t.Fatalf("harus melaporkan konfigurasi tidak lengkap, dapat: %s", keluaran)
	}
}
