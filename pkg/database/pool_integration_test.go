//go:build integration

package database_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.internal/business-data-api/pkg/database"
)

func nilaiEnv(t *testing.T, kunci string) string {
	t.Helper()
	if os.Getenv("INTEGRATION_DB") != "1" {
		t.Skip("INTEGRATION_DB != 1, lewati test DB")
	}
	isi, err := os.ReadFile(filepath.Join("..", "..", ".env"))
	if err != nil {
		t.Skipf("berkas .env tidak terbaca, lewati test DB: %v", err)
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
	t.Skipf("kunci %s tidak ada di .env", kunci)
	return ""
}

// Kedua pool diuji hanya sampai Ping. Tidak ada satu pun SELECT, INSERT,
// UPDATE, atau DELETE yang dikirim ke server.
func TestPoolPostgresSungguhanTerbukaDanDapatDiping(t *testing.T) {
	dsn := nilaiEnv(t, "POSTGRES_MAXTOP_URL")

	ctx, batal := context.WithTimeout(context.Background(), 30*time.Second)
	defer batal()

	pool, err := database.NewPostgresPool(ctx, dsn, database.PostgresPoolOptions{
		MaxConns:          2,
		MinConns:          1,
		MaxConnIdleTime:   time.Minute,
		MaxConnLifetime:   5 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("pool postgres harus terbuka: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("pool postgres harus bisa di-ping: %v", err)
	}
	if pool.Stat().MaxConns() != 2 {
		t.Fatalf("batas koneksi tak diterapkan: %+v", pool.Stat())
	}
}

func TestPoolMSSQLSungguhanTerbukaDanDapatDiping(t *testing.T) {
	dsn := nilaiEnv(t, "MSSQL_MAXTOP_URL")

	db, err := database.NewMSSQLDB(context.Background(), dsn, database.MSSQLPoolOptions{
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		MaxConnLifetime: 5 * time.Minute,
	})
	if err != nil {
		t.Fatalf("db mssql harus terbuka: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("db mssql harus bisa di-ping: %v", err)
	}
	if got := db.Stats().MaxOpenConnections; got != 2 {
		t.Fatalf("batas koneksi tak diterapkan: %d", got)
	}
}
