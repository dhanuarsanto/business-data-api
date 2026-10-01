package main

import (
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

const penandaSubproses = "JWTGEN_SUBPROSES"

// main memanggil os.Exit(1) saat APP_ENV=production, jadi jalur itu hanya bisa
// diuji lewat proses anak. Proses anak membaca penanda yang sama dengan GOCOVERDIR
// sehingga cakupan dari main() tetap digabung ke profil utama.
func TestMain(m *testing.M) {
	if os.Getenv(penandaSubproses) == "1" {
		main()
		return
	}
	os.Exit(m.Run())
}

func tangkapKeluaran(t *testing.T, jalankan func()) string {
	t.Helper()
	berkas, err := os.CreateTemp(t.TempDir(), "keluaran")
	if err != nil {
		t.Fatalf("buat berkas sementara gagal: %v", err)
	}
	asal := os.Stdout
	os.Stdout = berkas
	t.Cleanup(func() { os.Stdout = asal })

	jalankan()

	if err := berkas.Close(); err != nil {
		t.Fatalf("tutup berkas keluaran gagal: %v", err)
	}
	isi, err := os.ReadFile(berkas.Name())
	if err != nil {
		t.Fatalf("baca keluaran gagal: %v", err)
	}
	return string(isi)
}

var polaHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestMainMencetakSecret32ByteYangValid(t *testing.T) {
	t.Setenv("APP_ENV", "development")

	keluaran := tangkapKeluaran(t, main)

	if !strings.Contains(keluaran, "JWT_SECRET") {
		t.Fatalf("keluaran harus memandu penempatan JWT_SECRET, dapat: %s", keluaran)
	}
	fields := strings.Fields(keluaran)
	if len(fields) == 0 {
		t.Fatalf("keluaran kosong: %s", keluaran)
	}
	secret := fields[len(fields)-1]
	if !polaHex64.MatchString(secret) {
		t.Fatalf("secret harus 64 karakter heksadesimal, dapat %q", secret)
	}
	if _, err := hex.DecodeString(secret); err != nil {
		t.Fatalf("secret bukan heksadesimal valid: %v", err)
	}
}

func TestMainMenghasilkanSecretBerbedaSetiapPemanggilan(t *testing.T) {
	t.Setenv("APP_ENV", "development")

	ambil := func() string {
		fields := strings.Fields(tangkapKeluaran(t, main))
		return fields[len(fields)-1]
	}

	if ambil() == ambil() {
		t.Fatal("dua pemanggilan harus menghasilkan secret yang berbeda")
	}
}

func TestMainMenolakDijalankanDiProduksi(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$", "-test.count=1")
	cmd.Env = append(os.Environ(), penandaSubproses+"=1", "APP_ENV=production")
	keluaran, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("proses anak harus keluar dengan kode 1, dapat err=%v keluaran=%s", err, keluaran)
	}
	if !strings.Contains(string(keluaran), "AKSES DITOLAK") {
		t.Fatalf("proses anak harus menolak dengan jelas, dapat: %s", keluaran)
	}
	if polaHex64.MatchString(strings.TrimSpace(string(keluaran))) {
		t.Fatalf("proses anak tidak boleh mencetak secret: %s", keluaran)
	}
}
