package middleware

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyManagerLookupAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api_keys.json")

	if err := os.WriteFile(path, []byte(`{"key_a":"Dev A"}`), 0o644); err != nil {
		t.Fatalf("tulis file gagal: %v", err)
	}

	km := NewKeyManager(path)
	defer km.Stop()

	if name, ok := km.Lookup("key_a"); !ok || name != "Dev A" {
		t.Fatalf("Lookup key_a salah: ok=%v name=%q", ok, name)
	}
	if _, ok := km.Lookup("key_b"); ok {
		t.Fatal("key_b belum ada, harus tidak ditemukan")
	}

	if err := os.WriteFile(path, []byte(`{"key_a":"Dev A","key_b":"Dev B"}`), 0o644); err != nil {
		t.Fatalf("tulis ulang file gagal: %v", err)
	}
	if err := km.reload(); err != nil {
		t.Fatalf("reload gagal: %v", err)
	}
	if name, ok := km.Lookup("key_b"); !ok || name != "Dev B" {
		t.Fatalf("key_b setelah reload salah: ok=%v name=%q", ok, name)
	}
}

func TestKeyManagerKeepsOldOnCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "api_keys.json")

	if err := os.WriteFile(path, []byte(`{"key_a":"Dev A"}`), 0o644); err != nil {
		t.Fatalf("tulis file gagal: %v", err)
	}

	km := NewKeyManager(path)
	defer km.Stop()

	if err := os.WriteFile(path, []byte(`bukan-json`), 0o644); err != nil {
		t.Fatalf("tulis file corrupt gagal: %v", err)
	}
	if err := km.reload(); err == nil {
		t.Fatal("reload file corrupt harus error")
	}
	if name, ok := km.Lookup("key_a"); !ok || name != "Dev A" {
		t.Fatalf("key lama harus dipertahankan: ok=%v name=%q", ok, name)
	}
}
