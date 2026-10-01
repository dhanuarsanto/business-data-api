package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const penandaSubproses = "KEYGEN_SUBPROSES"

var (
	polaKunci  = regexp.MustCompile(`^key_[0-9a-f]{64}$`)
	polaBawaan = regexp.MustCompile(`^Gunakan perintah: go run cmd/keygen/main\.go`)
)

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

func denganArgumen(t *testing.T, argumen ...string) {
	t.Helper()
	asal := os.Args
	os.Args = append([]string{"keygen"}, argumen...)
	t.Cleanup(func() { os.Args = asal })
}

func bacaKunci(t *testing.T, path string) map[string]string {
	t.Helper()
	isi, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("baca %s gagal: %v", path, err)
	}
	kunci := map[string]string{}
	if err := json.Unmarshal(isi, &kunci); err != nil {
		t.Fatalf("urai %s gagal: %v", path, err)
	}
	return kunci
}

func TestMainMenampilkanPetunjukTanpaArgumen(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	denganArgumen(t)

	keluaran := tangkapKeluaran(t, main)

	if !polaBawaan.MatchString(keluaran) {
		t.Fatalf("harus menampilkan cara pemakaian, dapat: %s", keluaran)
	}
}

func TestMainMembuatApiKeyBaruPadaBerkasKosong(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	path := filepath.Join(t.TempDir(), "api_keys.json")
	t.Setenv("API_KEYS_PATH", path)
	denganArgumen(t, "Budi Santoso")

	keluaran := tangkapKeluaran(t, main)

	kunci := bacaKunci(t, path)
	if len(kunci) != 1 {
		t.Fatalf("harus tepat satu kunci, dapat %d", len(kunci))
	}
	var baru string
	for k, v := range kunci {
		baru = k
		if v != "Budi Santoso" {
			t.Fatalf("nama developer tidak sesuai: %q", v)
		}
	}
	if !polaKunci.MatchString(baru) {
		t.Fatalf("format kunci tidak sesuai: %q", baru)
	}
	if _, err := hex.DecodeString(strings.TrimPrefix(baru, "key_")); err != nil {
		t.Fatalf("bagian heksadesimal kunci tidak valid: %v", err)
	}
	if !strings.Contains(keluaran, "Budi Santoso") || !strings.Contains(keluaran, baru) {
		t.Fatalf("keluaran harus menyebut developer dan kunci, dapat: %s", keluaran)
	}
}

func TestMainMenambahKunciTanpaMenghapusYangLama(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	path := filepath.Join(t.TempDir(), "api_keys.json")
	t.Setenv("API_KEYS_PATH", path)
	lama := "key_" + strings.Repeat("a", 64)
	awal := map[string]string{lama: "Siti Aminah"}
	isi, err := json.MarshalIndent(awal, "", "  ")
	if err != nil {
		t.Fatalf("marshal awal gagal: %v", err)
	}
	if err := os.WriteFile(path, isi, 0o600); err != nil {
		t.Fatalf("tulis awal gagal: %v", err)
	}
	denganArgumen(t, "Andi")

	tangkapKeluaran(t, main)

	kunci := bacaKunci(t, path)
	if len(kunci) != 2 {
		t.Fatalf("harus dua kunci, dapat %d: %v", len(kunci), kunci)
	}
	if kunci[lama] != "Siti Aminah" {
		t.Fatalf("kunci lama hilang atau berubah: %v", kunci)
	}
	for k, v := range kunci {
		if k == lama {
			continue
		}
		if !polaKunci.MatchString(k) || v != "Andi" {
			t.Fatalf("kunci baru tidak sesuai: %q -> %q", k, v)
		}
	}
}

func TestMainMengabaikanIsiBerkasYangTidakValid(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	path := filepath.Join(t.TempDir(), "api_keys.json")
	t.Setenv("API_KEYS_PATH", path)
	if err := os.WriteFile(path, []byte("bukan json"), 0o600); err != nil {
		t.Fatalf("tulis awal gagal: %v", err)
	}
	denganArgumen(t, "Rina")

	tangkapKeluaran(t, main)

	kunci := bacaKunci(t, path)
	if len(kunci) != 1 {
		t.Fatalf("berkas rusak harus ditimpa bersih jadi satu kunci, dapat: %v", kunci)
	}
	for k, v := range kunci {
		if !polaKunci.MatchString(k) || v != "Rina" {
			t.Fatalf("kunci hasil tidak sesuai: %q -> %q", k, v)
		}
	}
}

func chdirSementara(t *testing.T, dir string) {
	t.Helper()
	asal, err := os.Getwd()
	if err != nil {
		t.Fatalf("ambil direktori kerja gagal: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("pindah direktori kerja gagal: %v", err)
	}
	t.Cleanup(func() { os.Chdir(asal) })
}

func TestMainMemakaiApiKeysJsonSaatPathKosong(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("API_KEYS_PATH", "")
	dir := t.TempDir()
	chdirSementara(t, dir)
	denganArgumen(t, "Dewi")

	tangkapKeluaran(t, main)

	kunci := bacaKunci(t, filepath.Join(dir, "api_keys.json"))
	if len(kunci) != 1 {
		t.Fatalf("harus tepat satu kunci pada api_keys.json bawaan, dapat: %v", kunci)
	}
	for k, v := range kunci {
		if !polaKunci.MatchString(k) || v != "Dewi" {
			t.Fatalf("kunci hasil tidak sesuai: %q -> %q", k, v)
		}
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
	if strings.Contains(string(keluaran), "API Key:") {
		t.Fatalf("proses anak tidak boleh mencetak kunci: %s", keluaran)
	}
}
