package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"go.internal/business-data-api/pkg/logger"
)

func TestRegistryMenyimpanDanMengambilKoneksi(t *testing.T) {
	r := NewDBRegistry()

	if _, err := r.Postgres("maxtop"); !errors.Is(err, ErrTenantNotFound) {
		t.Fatalf("tenant tak terdaftar harus ErrTenantNotFound, dapat %v", err)
	}
	if _, err := r.MSSQL("maxtop"); !errors.Is(err, ErrTenantNotFound) {
		t.Fatalf("tenant tak terdaftar harus ErrTenantNotFound, dapat %v", err)
	}

	pool := &pgxpool.Pool{}
	db, err := sql.Open("mssql-logged", "sqlserver://127.0.0.1:1/tidak-ada")
	if err != nil {
		t.Fatalf("sql.Open tak boleh gagal sebelum ping: %v", err)
	}
	defer db.Close()

	r.Register("maxtop", pool, db)

	gotPG, err := r.Postgres("maxtop")
	if err != nil || gotPG != pool {
		t.Fatalf("pool postgres harus dikembalikan utuh: %v %v", gotPG, err)
	}
	gotMS, err := r.MSSQL("maxtop")
	if err != nil || gotMS != db {
		t.Fatalf("db mssql harus dikembalikan utuh: %v %v", gotMS, err)
	}

	pengganti := &pgxpool.Pool{}
	r.Register("maxtop", pengganti, nil)
	if got, _ := r.Postgres("maxtop"); got != pengganti {
		t.Fatal("registrasi ulang harus menimpa koneksi lama")
	}
	if got, err := r.MSSQL("maxtop"); err != nil || got != nil {
		t.Fatalf("koneksi mssql harus jadi nil, dapat %v %v", got, err)
	}
}

func TestCloseAllMenutupKoneksiTanpaPanik(t *testing.T) {
	r := NewDBRegistry()
	db, err := sql.Open("mssql-logged", "sqlserver://127.0.0.1:1/tidak-ada")
	if err != nil {
		t.Fatalf("sql.Open tak boleh gagal sebelum ping: %v", err)
	}
	r.Register("tanpa-pg", nil, db)
	r.Register("tanpa-keduanya", nil, nil)

	r.CloseAll()

	if err := db.Ping(); err == nil {
		t.Fatal("koneksi harus tertutup setelah CloseAll")
	}
	r.CloseAll()
}

func TestSlogLoggerHanyaMenulisQueryDanMembacaDurasi(t *testing.T) {
	l := &SlogLogger{}
	ctx := logger.SetTraceContext(context.Background(), &logger.TraceContext{TraceID: "tr-1"})

	l.Log(ctx, tracelog.LogLevelInfo, "Koneksi dibuat", map[string]any{})
	l.Log(ctx, tracelog.LogLevelInfo, "Query", map[string]any{"sql": "SELECT 1", "time": 2 * time.Millisecond})
	l.Log(ctx, tracelog.LogLevelInfo, "Query", map[string]any{"sql": "SELECT 2"})
	l.Log(ctx, tracelog.LogLevelInfo, "Query", map[string]any{"sql": "SELECT 3", "time": slowQueryThreshold + time.Second})
	l.Log(ctx, tracelog.LogLevelInfo, "Query", map[string]any{"sql": "SELECT 4", "time": "bukan-duration"})
}

func TestLogMSSQLQueryMemisahkanCepatDanLambat(t *testing.T) {
	ctx := logger.SetTraceContext(context.Background(), &logger.TraceContext{TraceID: "tr-2"})

	logMSSQLQuery(ctx, "SELECT cepat", time.Now())
	logMSSQLQuery(ctx, "SELECT lambat", time.Now().Add(-slowQueryThreshold-time.Second))
}

func TestNewPostgresPoolMenerapkanSeluruhOpsiDanMenutupPoolGagal(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	const connString = "postgres://user:rahasia@127.0.0.1:1/tidak-ada?connect_timeout=1"
	opts := PostgresPoolOptions{
		MaxConns:          3,
		MinConns:          1,
		MaxConnIdleTime:   time.Minute,
		MaxConnLifetime:   2 * time.Minute,
		HealthCheckPeriod: 30 * time.Second,
	}

	pool, err := NewPostgresPool(ctx, connString, opts)
	if err == nil {
		pool.Close()
		t.Fatal("server tak terjangkau harus menghasilkan error")
	}

	if !strings.Contains(err.Error(), "tidak dapat di-ping") {
		t.Fatalf("error harus menunjuk ping, dapat %v", err)
	}
	if pool != nil {
		pool.Close()
		t.Fatal("pool tak boleh dikembalikan saat ping gagal")
	}
}

func TestNewPostgresPoolTanpaOpsiMemakaiBawaan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := NewPostgresPool(ctx, "postgres://user:rahasia@127.0.0.1:1/tidak-ada?connect_timeout=1",
		PostgresPoolOptions{})
	if err == nil {
		pool.Close()
		t.Fatal("server tak terjangkau harus menghasilkan error")
	}
}

func TestNewPostgresPoolMenolakURLRusak(t *testing.T) {
	if _, err := NewPostgresPool(context.Background(), "://bukan-url", PostgresPoolOptions{}); err == nil {
		t.Fatal("URL rusak harus ditolak")
	}
}

func TestNewMSSQLDBGagalSaatPingDanTutupKoneksi(t *testing.T) {
	const connString = "sqlserver://user:rahasia@127.0.0.1:1/tidak-ada?dial+timeout=1&connection+timeout=1"

	db, err := NewMSSQLDB(context.Background(), connString,
		MSSQLPoolOptions{MaxOpenConns: 4, MaxIdleConns: 2, MaxConnLifetime: time.Minute})
	if err == nil {
		db.Close()
		t.Fatal("server tak terjangkau harus menghasilkan error")
	}
	if !strings.Contains(err.Error(), "tidak dapat di-ping") {
		t.Fatalf("error harus menunjuk ping, dapat %v", err)
	}
}

func TestNewMSSQLDBTanpaOpsiMemakaiBawaan(t *testing.T) {
	db, err := NewMSSQLDB(context.Background(),
		"sqlserver://user:rahasia@127.0.0.1:1/tidak-ada?dial+timeout=1&connection+timeout=1",
		MSSQLPoolOptions{})
	if err == nil {
		db.Close()
		t.Fatal("server tak terjangkau harus menghasilkan error")
	}
}

func TestNewMSSQLDBMenolakConnStringTakTerurai(t *testing.T) {
	db, err := NewMSSQLDB(context.Background(), "sqlserver://user:pw@host:bukanport/db", MSSQLPoolOptions{})
	if err == nil {
		db.Close()
		t.Fatal("port tak valid harus ditolak saat membuka koneksi")
	}
	if !strings.Contains(err.Error(), "gagal membuka koneksi MSSQL") {
		t.Fatalf("pesan error tak terduga: %v", err)
	}
}

func TestMSSQLLoggerMencatatKueriLewatDriverTerdaftar(t *testing.T) {
	db, err := sql.Open("mssql-logged", "sqlserver://user:rahasia@127.0.0.1:1/tidak-ada?dial+timeout=1")
	if err != nil {
		t.Fatalf("sql.Open harus berhasil: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(context.Background(), "SELECT 1"); err == nil {
		t.Fatal("exec ke server tak terjangkau harus gagal")
	}
	rows, err := db.QueryContext(context.Background(), "SELECT 1")
	if err == nil {
		rows.Close()
		t.Fatal("query ke server tak terjangkau harus gagal")
	}
}
