//go:build integration

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const kunciUji = "key_uji_alat_baja_9d2f"

type anakProses struct {
	cmd *exec.Cmd
	rp  *io.PipeReader
	wp  *io.PipeWriter

	mu   sync.Mutex
	teks strings.Builder
}

func akarRepo(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("gagal menentukan akar repo: %v", err)
	}
	return root
}

// ruangAnak menyalin .env ke direktori sementara. Dengan begitu godotenv.Load()
// di LoadConfig() membaca berkas itu, sementara logs/ dan api_keys.json tetap
// berada di direktori sementara dan tidak pernah menyentuh repo.
func ruangAnak(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	isiEnv, err := os.ReadFile(filepath.Join(akarRepo(t), ".env"))
	if err != nil {
		t.Skipf("berkas .env tidak terbaca, lewati test integrasi: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), isiEnv, 0o600); err != nil {
		t.Fatalf("salin .env gagal: %v", err)
	}

	kunci, err := json.Marshal(map[string]string{kunciUji: "Alat Uji"})
	if err != nil {
		t.Fatalf("marshal kunci gagal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "api_keys.json"), kunci, 0o600); err != nil {
		t.Fatalf("tulis api_keys.json gagal: %v", err)
	}
	return dir
}

func portBebas(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("cari port bebas gagal: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func mulaiAnak(t *testing.T, dir string, port int, tambahan map[string]string) *anakProses {
	t.Helper()

	isi := map[string]string{"PORT": fmt.Sprint(port)}
	for k, v := range tambahan {
		isi[k] = v
	}

	rp, wp := io.Pipe()
	a := &anakProses{rp: rp, wp: wp}
	a.cmd = exec.Command(os.Args[0], "-test.run=^$", "-test.count=1")
	a.cmd.Dir = dir
	a.cmd.Env = lingkunganAnak(t, isi)
	a.cmd.Stdout = wp
	a.cmd.Stderr = wp
	SiapkanSinyal(a.cmd)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := rp.Read(buf)
			if n > 0 {
				a.mu.Lock()
				a.teks.Write(buf[:n])
				a.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()

	if err := a.cmd.Start(); err != nil {
		t.Fatalf("gagal memulai anak proses: %v", err)
	}
	t.Cleanup(func() {
		if a.cmd.ProcessState == nil {
			_ = a.cmd.Process.Kill()
		}
	})
	return a
}

func (a *anakProses) keluaran() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.teks.String()
}

func (a *anakProses) tutup(batas time.Duration) (int, error) {
	selesai := make(chan error, 1)
	go func() { selesai <- a.cmd.Wait() }()

	select {
	case err := <-selesai:
		a.wp.Close()
		a.rp.Close()
		var exitErr *exec.ExitError
		switch {
		case err == nil:
			return 0, nil
		case errors.As(err, &exitErr):
			return exitErr.ExitCode(), nil
		default:
			return -1, err
		}
	case <-time.After(batas):
		_ = a.cmd.Process.Kill()
		<-selesai
		a.wp.Close()
		a.rp.Close()
		return -1, fmt.Errorf("proses anak tidak keluar dalam %s", batas)
	}
}

func (a *anakProses) hentikanDanTunggu(t *testing.T, port int, batas time.Duration) {
	t.Helper()
	tungguSehat(t, port, a, 60*time.Second)
	if err := KirimSinyalAndal(a.cmd.Process); err != nil {
		t.Fatalf("gagal mengirim sinyal berhenti: %v", err)
	}
	kode, err := a.tutup(batas)
	if err != nil {
		t.Fatalf("%v\nkeluaran:\n%s", err, a.keluaran())
	}
	if kode != 0 {
		t.Fatalf("shutdown bersih harus keluar dengan kode 0, dapat %d\nkeluaran:\n%s", kode, a.keluaran())
	}
}

func tungguSehat(t *testing.T, port int, a *anakProses, batas time.Duration) {
	t.Helper()
	klien := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/health", port)

	var terakhir error
	for batasTunggu := time.Now().Add(batas); time.Now().Before(batasTunggu); {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatalf("bentuk permintaan gagal: %v", err)
		}
		req.Header.Set("X-API-KEY", kunciUji)
		resp, err := klien.Do(req)
		if err != nil {
			terakhir = err
			time.Sleep(150 * time.Millisecond)
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return
		}
		terakhir = fmt.Errorf("status %d", resp.StatusCode)
		time.Sleep(150 * time.Millisecond)
	}
	t.Fatalf("server tidak pernah siap di %s: %v\nkeluaran:\n%s", url, terakhir, a.keluaran())
}

func siapIntegrasi(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION_DB") != "1" {
		t.Skip("INTEGRATION_DB != 1, lewati test integrasi")
	}
}

func wajibAda(t *testing.T, a *anakProses, fragmen ...string) {
	t.Helper()
	keluaran := a.keluaran()
	for _, f := range fragmen {
		if !strings.Contains(keluaran, f) {
			t.Fatalf("keluaran tidak memuat %q\nkeluaran:\n%s", f, keluaran)
		}
	}
}

// nilaiEnv membaca satu kunci dari .env yang sudah disalin ke direktori anak,
// supaya tidak ada string koneksi maupun kredensial yang tersimpan literal di repo.
func nilaiEnv(t *testing.T, dir, kunci string) string {
	t.Helper()
	isi, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatalf("baca .env gagal: %v", err)
	}
	for _, baris := range strings.Split(string(isi), "\n") {
		baris = strings.TrimSpace(baris)
		if baris == "" || strings.HasPrefix(baris, "#") {
			continue
		}
		nama, nilai, ok := strings.Cut(baris, "=")
		if ok && strings.TrimSpace(nama) == kunci {
			return strings.Trim(strings.TrimSpace(nilai), `"`)
		}
	}
	t.Fatalf("kunci %s tidak ada di .env", kunci)
	return ""
}

// portMati mempertahankan host, kredensial, dan nama database dari DSN asli,
// hanya memindahkan port ke 1 yang tidak pernah mendengarkan, jadi tidak ada
// soket yang sampai ke server mana pun.
func portMati(t *testing.T, dsn string) string {
	t.Helper()
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("urai DSN gagal: %v", err)
	}
	u.Host = net.JoinHostPort(u.Hostname(), "1")
	return u.String()
}

func TestServerJalanLaluMatikanBersihLewatSinyal(t *testing.T) {
	siapIntegrasi(t)
	dir := ruangAnak(t)
	port := portBebas(t)
	a := mulaiAnak(t, dir, port, nil)

	a.hentikanDanTunggu(t, port, 45*time.Second)

	wajibAda(t, a,
		"Menjalankan API",
		"Sinyal berhenti diterima, mematikan server...",
		"Server berhasil dimatikan dengan aman",
	)
	if _, err := os.Stat(filepath.Join(dir, "logs", "api.log")); err != nil {
		t.Fatalf("logger harus menulis log di direktori kerja anak: %v", err)
	}
	if conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond); err == nil {
		conn.Close()
		t.Fatalf("port %d masih terbuka, server bocor", port)
	}
}

func TestServerGagalBerdengarSaatPortSudahDipakai(t *testing.T) {
	siapIntegrasi(t)
	dir := ruangAnak(t)
	port := portBebas(t)

	penadah, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("siapkan penadah port gagal: %v", err)
	}
	defer penadah.Close()

	a := mulaiAnak(t, dir, port, nil)
	kode, err := a.tutup(60 * time.Second)
	if err != nil {
		t.Fatalf("%v\nkeluaran:\n%s", err, a.keluaran())
	}
	if kode != 1 {
		t.Fatalf("port bentrok harus keluar dengan kode 1, dapat %d\nkeluaran:\n%s", kode, a.keluaran())
	}
	wajibAda(t, a, "Kesalahan server")
}

func TestServerMenolakProxyTidakValid(t *testing.T) {
	siapIntegrasi(t)
	dir := ruangAnak(t)
	port := portBebas(t)
	a := mulaiAnak(t, dir, port, map[string]string{"TRUSTED_PROXIES": "bukan-alamat"})

	kode, err := a.tutup(60 * time.Second)
	if err != nil {
		t.Fatalf("%v\nkeluaran:\n%s", err, a.keluaran())
	}
	if kode != 1 {
		t.Fatalf("proxy tak valid harus keluar dengan kode 1, dapat %d\nkeluaran:\n%s", kode, a.keluaran())
	}
	wajibAda(t, a, "TRUSTED_PROXIES tidak valid")
	if strings.Contains(a.keluaran(), "TRUSTED_PROXIES kosong") {
		t.Fatalf("jalur proxy kosong tidak boleh ikut terpicu\nkeluaran:\n%s", a.keluaran())
	}
}

func TestServerMenolakProduksiTanpaJalurKlien(t *testing.T) {
	siapIntegrasi(t)
	dir := ruangAnak(t)
	port := portBebas(t)
	a := mulaiAnak(t, dir, port, map[string]string{
		"APP_ENV":              "production",
		"TRUSTED_PROXIES":      "",
		"GLOBAL_LOCAL_ONLY":    "true",
		"ALLOW_DIRECT_CLIENTS": "false",
	})

	kode, err := a.tutup(60 * time.Second)
	if err != nil {
		t.Fatalf("%v\nkeluaran:\n%s", err, a.keluaran())
	}
	if kode != 1 {
		t.Fatalf("produksi tanpa jalur klien harus keluar dengan kode 1, dapat %d\nkeluaran:\n%s", kode, a.keluaran())
	}
	wajibAda(t, a, "GLOBAL_LOCAL_ONLY=true")
	if strings.Contains(a.keluaran(), "TRUSTED_PROXIES kosong") {
		t.Fatalf("peringatan proxy kosong tidak boleh muncul setelah penolakan\nkeluaran:\n%s", a.keluaran())
	}
}

func TestServerTetapJalanDenganProxyKosongDiNonProduksi(t *testing.T) {
	siapIntegrasi(t)
	dir := ruangAnak(t)
	port := portBebas(t)
	a := mulaiAnak(t, dir, port, map[string]string{"TRUSTED_PROXIES": ""})

	a.hentikanDanTunggu(t, port, 45*time.Second)

	wajibAda(t, a, "TRUSTED_PROXIES kosong")
}

func TestServerBerhentiSaatSatuTenantTidakTerhubung(t *testing.T) {
	kasus := []struct {
		tenant string
		kunci  string
		pesan  string
	}{
		{"MSSQL Maxtop", "MSSQL_MAXTOP_URL", "Gagal koneksi MSSQL Maxtop"},
		{"Postgres Pandora", "POSTGRES_PANDORA_URL", "Gagal koneksi Postgres Pandora"},
		{"MSSQL Pandora", "MSSQL_PANDORA_URL", "Gagal koneksi MSSQL Pandora"},
		{"Postgres Toplink", "POSTGRES_TOPLINK_URL", "Gagal koneksi Postgres Toplink"},
		{"MSSQL Toplink", "MSSQL_TOPLINK_URL", "Gagal koneksi MSSQL Toplink"},
	}

	for _, k := range kasus {
		t.Run(k.tenant, func(t *testing.T) {
			siapIntegrasi(t)
			dir := ruangAnak(t)
			port := portBebas(t)
			a := mulaiAnak(t, dir, port, map[string]string{
				k.kunci: portMati(t, nilaiEnv(t, dir, k.kunci)),
			})

			kode, err := a.tutup(60 * time.Second)
			if err != nil {
				t.Fatalf("%v\nkeluaran:\n%s", err, a.keluaran())
			}
			if kode != 1 {
				t.Fatalf("harus keluar dengan kode 1, dapat %d\nkeluaran:\n%s", kode, a.keluaran())
			}
			wajibAda(t, a, k.pesan)
		})
	}
}

func TestServerMelaporkanPaksaMatiSaatShutdownLewatWaktu(t *testing.T) {
	siapIntegrasi(t)
	dir := ruangAnak(t)
	port := portBebas(t)
	a := mulaiAnak(t, dir, port, map[string]string{"SERVER_SHUTDOWN_TIMEOUT": "1s"})

	tungguSehat(t, port, a, 60*time.Second)

	// Koneksi dengan header HTTP yang belum lengkap membuat server considers
	// koneksi itu masih aktif, sehingga Shutdown menunggu sampai deadline.
	gantung, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 2*time.Second)
	if err != nil {
		t.Fatalf("buka koneksi gantung gagal: %v", err)
	}
	defer gantung.Close()
	if _, err := gantung.Write([]byte("GET /health HTTP/1.1\r\nHost: uji\r\n")); err != nil {
		t.Fatalf("tulis header sebagian gagal: %v", err)
	}

	if err := KirimSinyalAndal(a.cmd.Process); err != nil {
		t.Fatalf("gagal mengirim sinyal berhenti: %v", err)
	}
	kode, err := a.tutup(60 * time.Second)
	if err != nil {
		t.Fatalf("%v\nkeluaran:\n%s", err, a.keluaran())
	}
	if kode != 0 {
		t.Fatalf("kegagalan shutdown tidak boleh mengubah kode keluar, dapat %d\nkeluaran:\n%s", kode, a.keluaran())
	}
	wajibAda(t, a, "Server dipaksa mati", "Server berhasil dimatikan dengan aman")
}
