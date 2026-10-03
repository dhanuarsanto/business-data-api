package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"go.internal/business-data-api/pkg/database"
)

const (
	fakeDriverName = "mssql-fake-uji"
	fakeTenant     = "tenant-fake"
)

var (
	errPGNotScripted = errors.New("kueri Postgres tidak di-skrip")
	errMSNotScripted = errors.New("kueri MSSQL tidak di-skrip")
	errScanMismatch  = errors.New("tipe kolom tidak cocok dengan tujuan scan")
	errStreamBroken  = errors.New("koneksi terputus saat membaca baris")
)

// ---------------------------------------------------------------- PostgreSQL

type pgFake struct {
	mu          sync.Mutex
	queryFn     func(q string, args []any) (pgx.Rows, error)
	queryRowFn  func(q string, args []any) pgx.Row
	execFn      func(q string, args []any) (pgconn.CommandTag, error)
	queries     []string
	rowsQueries []string
	execs       []string
	closed      int
}

var _ database.PGConn = (*pgFake)(nil)

func (p *pgFake) Query(_ context.Context, q string, args ...any) (pgx.Rows, error) {
	p.mu.Lock()
	p.queries = append(p.queries, q)
	fn := p.queryFn
	p.mu.Unlock()
	if fn == nil {
		return nil, errPGNotScripted
	}
	return fn(q, args)
}

func (p *pgFake) QueryRow(_ context.Context, q string, args ...any) pgx.Row {
	p.mu.Lock()
	p.rowsQueries = append(p.rowsQueries, q)
	fn := p.queryRowFn
	p.mu.Unlock()
	if fn == nil {
		return &pgRowFake{scanErr: errPGNotScripted}
	}
	return fn(q, args)
}

func (p *pgFake) Exec(_ context.Context, q string, args ...any) (pgconn.CommandTag, error) {
	p.mu.Lock()
	p.execs = append(p.execs, q)
	fn := p.execFn
	p.mu.Unlock()
	if fn == nil {
		return pgconn.CommandTag{}, errPGNotScripted
	}
	return fn(q, args)
}

func (p *pgFake) Close() {
	p.mu.Lock()
	p.closed++
	p.mu.Unlock()
}

func (p *pgFake) queryLog() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.queries...)
}

func (p *pgFake) rowLog() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.rowsQueries...)
}

type pgRowsFake struct {
	values    []any
	remaining int
	iterErr   error
	scanErr   error
	closed    int
}

func (r *pgRowsFake) Next() bool {
	if r.remaining > 0 {
		r.remaining--
		return true
	}
	return false
}

func (r *pgRowsFake) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	return assignRow(dest, r.values)
}

func (r *pgRowsFake) Err() error { return r.iterErr }
func (r *pgRowsFake) Close()     { r.closed++ }
func (r *pgRowsFake) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 0")
}
func (r *pgRowsFake) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *pgRowsFake) Values() ([]any, error)                       { return r.values, nil }
func (r *pgRowsFake) RawValues() [][]byte                          { return nil }
func (r *pgRowsFake) Conn() *pgx.Conn                              { return nil }
func (r *pgRowsFake) TypeMap() *pgtype.Map                         { return nil }

type pgRowFake struct {
	values  []any
	scanErr error
}

func (r *pgRowFake) Scan(dest ...any) error {
	if r.scanErr != nil {
		return r.scanErr
	}
	return assignRow(dest, r.values)
}

func assignRow(dest []any, values []any) error {
	if len(dest) != len(values) {
		return fmt.Errorf("%w: %d tujuan vs %d nilai", errScanMismatch, len(dest), len(values))
	}
	for i, d := range dest {
		if err := assignValue(d, values[i]); err != nil {
			return fmt.Errorf("%w: kolom ke-%d: %v", errScanMismatch, i, err)
		}
	}
	return nil
}

func assignValue(dest any, value any) error {
	target := reflect.ValueOf(dest)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return errors.New("tujuan bukan pointer yang valid")
	}
	if value == nil {
		return nil
	}
	src := reflect.ValueOf(value)
	if target.Type() == src.Type() {
		target.Set(src)
		return nil
	}

	elem := target.Elem()
	if elem.Kind() != reflect.Pointer {
		return fillValue(elem, src)
	}
	if src.Kind() == reflect.Pointer && src.IsNil() {
		return nil
	}
	if src.Kind() == reflect.Pointer {
		src = src.Elem()
	}
	allocated := reflect.New(elem.Type().Elem())
	if err := fillValue(allocated.Elem(), src); err != nil {
		return err
	}
	elem.Set(allocated)
	return nil
}

func fillValue(target, src reflect.Value) error {
	switch {
	case !src.IsValid():
		return nil
	case src.Type().AssignableTo(target.Type()):
		target.Set(src)
	case src.Type().ConvertibleTo(target.Type()):
		target.Set(src.Convert(target.Type()))
	default:
		return fmt.Errorf("tipe %s tidak bisa ke %s", src.Type(), target.Type())
	}
	return nil
}

// -------------------------------------------------------------------- MSSQL

type msFake struct {
	mu        sync.Mutex
	queryFn   func(q string, args []driver.NamedValue) (driver.Rows, error)
	execFn    func(q string, args []driver.NamedValue) (driver.Result, error)
	queries   []string
	execs     []string
	openCount int
}

var msActive atomic.Pointer[msFake]

type msDriverFake struct{}

func init() { sql.Register(fakeDriverName, msDriverFake{}) }

func (msDriverFake) Open(string) (driver.Conn, error) {
	if f := msActive.Load(); f != nil {
		f.mu.Lock()
		f.openCount++
		f.mu.Unlock()
	}
	return &msConnFake{}, nil
}

type msConnFake struct{}

func (c *msConnFake) Prepare(string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *msConnFake) Close() error                        { return nil }
func (c *msConnFake) Begin() (driver.Tx, error)           { return nil, errMSNotScripted }

func (c *msConnFake) CheckNamedValue(*driver.NamedValue) error { return nil }

func (c *msConnFake) QueryContext(_ context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	f := msActive.Load()
	if f == nil {
		return nil, errMSNotScripted
	}
	f.mu.Lock()
	f.queries = append(f.queries, q)
	fn := f.queryFn
	f.mu.Unlock()
	if fn == nil {
		return nil, errMSNotScripted
	}
	return fn(q, args)
}

func (c *msConnFake) ExecContext(_ context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	f := msActive.Load()
	if f == nil {
		return nil, errMSNotScripted
	}
	f.mu.Lock()
	f.execs = append(f.execs, q)
	fn := f.execFn
	f.mu.Unlock()
	if fn == nil {
		return nil, errMSNotScripted
	}
	return fn(q, args)
}

func (f *msFake) queryLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.queries...)
}

func (f *msFake) execLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.execs...)
}

type msRowsFake struct {
	columns []string
	data    [][]driver.Value
	pos     int
	failAt  int
	failErr error
}

func (r *msRowsFake) Columns() []string { return r.columns }
func (r *msRowsFake) Close() error      { return nil }

func (r *msRowsFake) Next(dest []driver.Value) error {
	if r.failErr != nil && r.pos >= r.failAt {
		return r.failErr
	}
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

// tipeTidakCocok dipakai untuk memicu error konversi saat pemindaian baris.
type tipeTidakCocok struct{ Kolom int }

type msResultFake struct{ affected int64 }

func (r msResultFake) LastInsertId() (int64, error) { return 0, errMSNotScripted }
func (r msResultFake) RowsAffected() (int64, error) { return r.affected, nil }

// ------------------------------------------------------------------ fixtures

type fakeEnv struct {
	reg *database.DBRegistry
	pg  *pgFake
	ms  *msFake
}

func newFakeEnv(t *testing.T) *fakeEnv {
	t.Helper()
	pg := &pgFake{}
	ms := &msFake{}
	msActive.Store(ms)

	db, err := sql.Open(fakeDriverName, "")
	if err != nil {
		t.Fatalf("gagal membuka sql.DB palsu: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	reg := database.NewDBRegistry()
	reg.Register(fakeTenant, pg, db)
	t.Cleanup(reg.CloseAll)
	return &fakeEnv{reg: reg, pg: pg, ms: ms}
}

func inboxRowValues(kode int64) []any {
	reseller := "RS-01"
	terminal := 7
	transaksi := 4242
	return []any{kode, ujiWaktu(), "0812345", reseller, "halo", int16(1), nil, terminal, "SC-1", transaksi}
}

func outboxRowValues(kode int64) []any {
	reseller := "RS-01"
	transaksi := 5151
	tipe := "1"
	var kodeInbox *int64
	return []any{kode, ujiWaktu(), "0812999", reseller, "hai", int16(0), nil, transaksi, tipe, kodeInbox}
}

func resellerRowValues(kode, nama string) []any {
	return []any{kode, nama}
}

func ujiWaktu() time.Time {
	return time.Date(2026, 9, 29, 10, 15, 0, 0, time.UTC)
}
