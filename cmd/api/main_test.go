package main

import (
	"testing"
	"time"

	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/pkg/database"
)

func TestPgPoolOptionsMenyalinSeluruhNilai(t *testing.T) {
	cfg := &config.Config{
		PostgresMaxConns:          7,
		PostgresMinConns:          2,
		PostgresMaxConnIdleTime:   3 * time.Minute,
		PostgresMaxConnLifetime:   11 * time.Minute,
		PostgresHealthCheckPeriod: 17 * time.Second,
	}

	got := pgPoolOptions(cfg)

	if got.MaxConns != 7 || got.MinConns != 2 {
		t.Fatalf("batas koneksi tak disalin: %+v", got)
	}
	if got.MaxConnIdleTime != 3*time.Minute || got.MaxConnLifetime != 11*time.Minute {
		t.Fatalf("usia koneksi tak disalin: %+v", got)
	}
	if got.HealthCheckPeriod != 17*time.Second {
		t.Fatalf("periode health check tak disalin: %+v", got)
	}
}

func TestMsPoolOptionsMenyalinSeluruhNilai(t *testing.T) {
	cfg := &config.Config{
		MSSQLMaxOpenConns:    9,
		MSSQLMaxIdleConns:    4,
		MSSQLMaxConnLifetime: 23 * time.Minute,
	}

	got := msPoolOptions(cfg)

	if got.MaxOpenConns != 9 || got.MaxIdleConns != 4 || got.MaxConnLifetime != 23*time.Minute {
		t.Fatalf("opsi MSSQL tak disalin: %+v", got)
	}
}

func TestBuildModulesMembangunSeluruhModulDanMatriks(t *testing.T) {
	cfg := &config.Config{CookieSecure: true}
	registry := database.NewDBRegistry()

	modules, networkMatrix, roleMatrix := BuildModules(registry, cfg, nil)

	if len(modules) != 4 {
		t.Fatalf("harus ada 4 modul, dapat %d", len(modules))
	}
	if networkMatrix["*"] != true || networkMatrix[roleSA] != false {
		t.Fatalf("matriks jaringan salah: %v", networkMatrix)
	}
	want := map[string][]string{
		"ManageUsers":          {roleSA},
		"ReadResellerDropdown": {roleSA, roleOP, roleOpOut},
		"ReadInbox":            {roleSA, roleOP, roleOpOut},
		"WriteInbox":           {roleSA},
		"ReadOutbox":           {roleSA, roleOP, roleOpOut},
		"WriteOutbox":          {roleSA},
	}
	for izin, peran := range want {
		got, ada := roleMatrix[izin]
		if !ada {
			t.Fatalf("izin %q tak terdaftar", izin)
		}
		if len(got) != len(peran) {
			t.Fatalf("izin %q: dapat %v, harus %v", izin, got, peran)
		}
		for i := range peran {
			if got[i] != peran[i] {
				t.Fatalf("izin %q indeks %d: %q harus %q", izin, i, got[i], peran[i])
			}
		}
	}
	if len(roleMatrix) != len(want) {
		t.Fatalf("izin tak terduga: %v", roleMatrix)
	}
}
