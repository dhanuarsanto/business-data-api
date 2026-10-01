package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type stubConn struct {
	dieks error
}

func (c *stubConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, c.dieks
}

func (c *stubConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return nil, c.dieks
}

type stubRows struct{}

func (stubRows) Columns() []string              { return []string{"kolom"} }
func (stubRows) Close() error                   { return nil }
func (stubRows) Next(dest []driver.Value) error { return io.EOF }

type stubResult struct{}

func (stubResult) LastInsertId() (int64, error) { return 0, nil }
func (stubResult) RowsAffected() (int64, error) { return 1, nil }

func TestMSSQLLoggerMeneruskanKonteksDanMencatatKueri(t *testing.T) {
	l := &mssqlLogger{}
	ctx := context.Background()

	balik, rows, err := l.ConnQueryContext(ctx, &stubConn{}, "SELECT 1", nil)
	if balik != ctx || rows != nil || err != nil {
		t.Fatalf("harus meneruskan hasil apa adanya: %v %v %v", rows == nil, err, balik == ctx)
	}

	res, err := l.ConnExecContext(ctx, &stubConn{}, "UPDATE t SET a=1", nil)
	if res != nil || err != nil {
		t.Fatalf("harus meneruskan hasil apa adanya: %v %v", res, err)
	}
}

func TestMSSQLLoggerMelewatiKueriBerhasilTanpaGangguan(t *testing.T) {
	l := &mssqlLogger{}
	ctx := context.Background()

	if _, rows, err := l.ConnQueryContext(ctx, &stubConnBerhasil{}, "SELECT 1", nil); err != nil || rows == nil {
		t.Fatalf("kueri berhasil harus diteruskan: %v %v", rows, err)
	}
	if _, err := l.ConnExecContext(ctx, &stubConnBerhasil{}, "DELETE FROM t", nil); err != nil {
		t.Fatalf("eksekusi berhasil tak boleh jadi error: %v", err)
	}
}

type stubConnBerhasil struct{}

func (*stubConnBerhasil) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return stubRows{}, nil
}

func (*stubConnBerhasil) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return stubResult{}, nil
}

func TestCloseAllMenutupPoolPostgresjuganya(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://user:rahasia@127.0.0.1:1/tidak-ada")
	if err != nil {
		t.Fatalf("parse config gagal: %v", err)
	}
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Skipf("pool tak bisa dibuat tanpa server: %v", err)
	}

	db, errBuka := sql.Open("mssql-logged", "sqlserver://user:rahasia@127.0.0.1:1/tidak-ada")
	if errBuka != nil {
		t.Fatalf("sql.Open gagal: %v", errBuka)
	}

	r := NewDBRegistry()
	r.Register("lengkap", pool, db)
	r.Register("hanya-pg", pool, nil)
	r.Register("hanya-ms", nil, db)
	r.Register("kosong", nil, nil)

	done := make(chan struct{})
	go func() {
		defer close(done)
		r.CloseAll()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("CloseAll menggantung")
	}

	if err := pool.Ping(context.Background()); err == nil {
		t.Fatal("pool postgres harus tertutup")
	}
	if err := db.Ping(); err == nil {
		t.Fatal("koneksi mssql harus tertutup")
	}
}
