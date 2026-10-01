package logger

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetTraceContextTanpaNilaiMemberiDefault(t *testing.T) {
	got := GetTraceContext(context.Background())

	if got == nil {
		t.Fatal("harus selalu mengembalikan struct")
	}
	if got.TraceID != "UNKNOWN" || got.Developer != "UNKNOWN" || got.IP != "UNKNOWN" || got.Path != "UNKNOWN" {
		t.Fatalf("default tidak lengkap: %+v", got)
	}
	if got.UserID != 0 {
		t.Fatalf("UserID default harus nol, dapat %d", got.UserID)
	}
}

func TestTraceContextBulatDiDalamContext(t *testing.T) {
	asal := &TraceContext{TraceID: "abc", Developer: "Budi"}
	ctx := SetTraceContext(context.Background(), asal)

	diambil := GetTraceContext(ctx)
	diambil.UserID = 42
	asal.Path = "/api/v1/maxtop/inbox"

	if asal.UserID != 42 {
		t.Fatal("perubahan harus terlihat pada struct asal")
	}
	if GetTraceContext(ctx).Path != "/api/v1/maxtop/inbox" {
		t.Fatal("penyimpanan harus memakai pointer yang sama")
	}
	if GetTraceContext(context.Background()) == diambil {
		t.Fatal("konteks tanpa nilai harus mengembalikan struct baru")
	}
}

func TestGetTraceContextAbaikanNilaiSalahTipe(t *testing.T) {
	type key struct{}
	ctx := context.WithValue(context.Background(), key{}, "bukan-struct")

	if got := GetTraceContext(ctx); got.TraceID != "UNKNOWN" {
		t.Fatalf("nilai salah tipe harus diabaikan, dapat %+v", got)
	}
}

func TestTeeHandlerMenulisKeKeduaSaluran(t *testing.T) {
	terminalFile := filepath.Join(t.TempDir(), "terminal.log")
	terminal, err := os.Create(terminalFile)
	if err != nil {
		t.Fatalf("buat berkas terminal gagal: %v", err)
	}
	defer terminal.Close()

	fileFile := filepath.Join(t.TempDir(), "file.log")
	berkas, err := os.Create(fileFile)
	if err != nil {
		t.Fatalf("buat berkas log gagal: %v", err)
	}
	defer berkas.Close()

	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	h := &TeeHandler{
		Terminal: slog.NewTextHandler(terminal, opts),
		File:     slog.NewJSONHandler(berkas, opts),
	}
	logger := slog.New(h)

	if !h.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("handler aktif harus enabled")
	}
	logger.Info("halo", "kunci", "nilai")
	if err := h.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "langsung", 0)); err != nil {
		t.Fatalf("Handle harus sukses: %v", err)
	}

	withAttrs := logger.With("a", 1).With("b", 2)
	withAttrs.Warn("dengan atribut")
	grouped := logger.WithGroup("grp")
	grouped.Error("dengan grup")

	terminal.Sync()
	berkas.Sync()

	for _, berkasUji := range []string{terminalFile, fileFile} {
		isi, err := os.ReadFile(berkasUji)
		if err != nil {
			t.Fatalf("baca %s gagal: %v", berkasUji, err)
		}
		teks := string(isi)
		if !strings.Contains(teks, "halo") || !strings.Contains(teks, "langsung") {
			t.Fatalf("%s tak berisi log dasar:\n%s", berkasUji, teks)
		}
		if !strings.Contains(teks, "dengan atribut") || !strings.Contains(teks, "dengan grup") {
			t.Fatalf("%s tak berisi log lanjutan:\n%s", berkasUji, teks)
		}
	}
}

func TestTeeHandlerEnabledMeloloskanSalahSatuSaluran(t *testing.T) {
	nonaktif := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})
	aktif := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})

	keduanya := &TeeHandler{Terminal: nonaktif, File: aktif}
	if !keduanya.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("salah satu saluran aktif harus cukup")
	}

	tidakAda := &TeeHandler{Terminal: nonaktif, File: nonaktif}
	if tidakAda.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("dua-duanya nonaktif harus disabled")
	}
}

func TestSetupLoggerMembuatBerkasDanMemakaiLevelLingkungan(t *testing.T) {
	// lumberjack menahan berkas log terbuka seumur proses, jadi direktori kerja
	// tidak bisa memakai t.TempDir: penghapusannya akan gagal di Windows.
	dir, err := os.MkdirTemp("", "uji-logger-")
	if err != nil {
		t.Fatalf("buat direktori uji gagal: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("berkas log masih terkunci, sisanya dibiarkan: %v", err)
		}
	})

	asal, err := os.Getwd()
	if err != nil {
		t.Fatalf("ambil cwd gagal: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir gagal: %v", err)
	}
	t.Cleanup(func() { os.Chdir(asal) })

	SetupLogger("test")
	slog.Debug("pesan debug harus ditulis")
	slog.Info("pesan info harus ditulis")

	deadline := time.Now().Add(2 * time.Second)
	var isi []byte
	for {
		isi, err = os.ReadFile(filepath.Join(dir, "logs", "api.log"))
		if err == nil && strings.Contains(string(isi), "pesan info harus ditulis") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("log tak tertulis: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(string(isi), "pesan debug harus ditulis") {
		t.Fatal("lingkungan non-produksi harus mengizinkan level debug")
	}

	SetupLogger("production")
	slog.Debug("debug produksi tak boleh bocor")
	slog.Info("info produksi harus bocor")

	deadline = time.Now().Add(2 * time.Second)
	for {
		isi, _ = os.ReadFile(filepath.Join(dir, "logs", "api.log"))
		if strings.Contains(string(isi), "info produksi harus bocor") {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("log produksi tak tertulis")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if strings.Contains(string(isi), "debug produksi tak boleh bocor") {
		t.Fatal("level produksi tak boleh membocorkan debug")
	}
}
