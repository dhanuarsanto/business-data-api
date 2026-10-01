package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const namaGuard = "db_readonly_guard_test.go"

type allowance struct {
	file string
	why  string
}

var allowances = []allowance{
	{
		"internal/config/load_config_test.go",
		"hanya mem-parsing string koneksi menjadi nilai Config; tidak ada soket dibuka",
	},
	{
		"internal/repository/fake_repo_test.go",
		"pernyataan tulis hanya muncul sebagai string pembanding dan diarahkan ke driver palsu in-memory; nol byte keluar dari proses",
	},
	{
		"internal/repository/readonly_integration_test.go",
		"satu-satunya test yang menyentuh server dan hanya menjalankan pembacaan; enam pemanggilan tulis memakai tenant tak terdaftar sehingga berhenti di registry sebelum SQL disusun",
	},
	{
		"pkg/database/database_test.go",
		"semua string koneksi menunjuk ke 127.0.0.1:1 yang ditolak; satu-satunya eksekusi menjalankan pembacaan terhadap port tertutup",
	},
	{
		"pkg/database/mssql_logger_test.go",
		"koneksi adalah stub in-memory milik logger driver, bukan server",
	},
	{
		"cmd/api/main_subproses_test.go",
		"seluruh string koneksi menunjuk ke 127.0.0.1:1 yang ditolak seketika; main() berhenti di NewPostgresPool sebelum query apa pun disusun",
	},
}

var writePatterns = []struct {
	label string
	ruang string
	re    *regexp.Regexp
}{
	{label: "penyisipan baris", re: regexp.MustCompile(`(?i)\bINSERT\s+INTO\b`)},
	{label: "pembaruan baris", re: regexp.MustCompile(`(?i)\bUPDATE\s+[A-Za-z_][A-Za-z0-9_]*\s+SET\b`)},
	{label: "penghapusan baris", re: regexp.MustCompile(`(?i)\bDELETE\s+FROM\b`)},
	{label: "penggabungan baris", re: regexp.MustCompile(`(?i)\bMERGE\s+INTO\b`)},
	{label: "pengosongan tabel", re: regexp.MustCompile(`(?i)\bTRUNCATE\b`)},
	{label: "penghapusan objek", re: regexp.MustCompile(`(?i)\bDROP\s+(TABLE|INDEX|DATABASE|SCHEMA|VIEW)\b`)},
	{label: "perubahan skema", re: regexp.MustCompile(`(?i)\bALTER\s+TABLE\b`)},
	{label: "pembuatan objek", re: regexp.MustCompile(`(?i)\bCREATE\s+(TABLE|INDEX|DATABASE|SCHEMA|UNIQUE)\b`)},
	{label: "pengaturan hak akses", re: regexp.MustCompile(`(?i)\b(GRANT|REVOKE)\b`)},
	{label: "eksekusi perintah", re: regexp.MustCompile(`\.(Exec|ExecContext|Begin|BeginTx)\(`)},
	{label: "string koneksi nyata", re: regexp.MustCompile(`(?i)(postgres|postgresql|sqlserver|mysql)://`)},
	{label: "pemanggilan tulis repository", ruang: "internal/repository/", re: regexp.MustCompile(`\.(Insert|Update|Delete)\(`)},
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("gagal menentukan akar repo: %v", err)
	}
	return root
}

func TestTidakAdaUjiYangMenulisDatabase(t *testing.T) {
	root := repoRoot(t)
	diizinkan := make(map[string]allowance, len(allowances))
	for _, a := range allowances {
		if a.why == "" {
			t.Fatalf("izin untuk %s wajib menyertakan alasan", a.file)
		}
		diizinkan[filepath.ToSlash(a.file)] = a
	}

	ditemukan := 0
	terpakai := make(map[string]bool, len(allowances))
	var pelanggaran []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "vendor", "logs", "bin":
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)

		if strings.HasSuffix(strings.ToLower(info.Name()), ".sql") {
			pelanggaran = append(pelanggaran, rel+": berkas .sql tidak boleh ada di repo")
			return nil
		}
		if !strings.HasSuffix(info.Name(), "_test.go") || info.Name() == namaGuard {
			return nil
		}

		isi, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		_, adaIzin := diizinkan[rel]
		for i, baris := range strings.Split(string(isi), "\n") {
			for _, p := range writePatterns {
				if p.ruang != "" && !strings.HasPrefix(rel, p.ruang) {
					continue
				}
				if !p.re.MatchString(baris) {
					continue
				}
				ditemukan++
				if adaIzin {
					terpakai[rel] = true
					continue
				}
				pelanggaran = append(pelanggaran, fmt.Sprintf("%s:%d: %s -> %s", rel, i+1, p.label, potong(baris, 120)))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("gagal memindai repo: %v", err)
	}

	if ditemukan == 0 {
		t.Fatal("pola deteksi tidak menemukan apa pun; aturan ini perlu diperbarui")
	}
	for rel := range diizinkan {
		if !terpakai[rel] {
			pelanggaran = append(pelanggaran, rel+": izin sudah tidak relevan karena tidak ada lagi pola yang cocok; hapus dari daftar")
		}
	}

	if len(pelanggaran) > 0 {
		t.Fatalf("ditemukan %d pelanggaran aturan baca-saja:\n%s", len(pelanggaran), strings.Join(pelanggaran, "\n"))
	}
}

func potong(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
