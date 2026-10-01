package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/pkg/database"
)

var (
	errScripted = errors.New("kegagalan basis data yang disimulasikan")
	tenantUji   = fakeTenant
)

func inboxFilterUji() domain.InboxFilter {
	return domain.InboxFilter{PageSize: 1, LimitTotal: ptr(10)}
}

func outboxFilterUji() domain.OutboxFilter {
	return domain.OutboxFilter{PageSize: 1, LimitTotal: ptr(10)}
}

func sampleInbox() domain.Inbox {
	return domain.Inbox{
		Pengirim:     "0812000",
		TipePengirim: "hp",
		Penerima:     ptr("0813999"),
		Pesan:        "isi pesan",
		Status:       0,
		KodeTerminal: ptr(3),
	}
}

func sampleOutbox() domain.Outbox {
	return domain.Outbox{
		Penerima:     "0813999",
		TipePenerima: "hp",
		Pesan:        "isi pesan",
		Status:       1,
		BebasBiaya:   0,
	}
}

func updateInboxRequestUji() dto.UpdateInboxRequest {
	return dto.UpdateInboxRequest{Status: ptr(int16(2)), Pesan: ptr("diperbarui")}
}

func updateOutboxRequestUji() dto.UpdateOutboxRequest {
	return dto.UpdateOutboxRequest{Status: ptr(int16(2)), Pesan: ptr("diperbarui")}
}

// ------------------------------------------------------------- Inbox: PG

func TestInboxPGGetBerhasilDenganHalamanBerikutnya(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: inboxRowValues(11), remaining: 2}, nil
	}

	items, hasNext, err := NewInboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, inboxFilterUji())
	if err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if !hasNext {
		t.Fatal("hasNextPage harus true")
	}
	if len(items) != 1 {
		t.Fatalf("harus 1 item setelah dipotong, dapat %d", len(items))
	}
	if items[0].Kode != 11 {
		t.Fatalf("kode tidak sesuai: %d", items[0].Kode)
	}
	if items[0].KodeReseller == nil || *items[0].KodeReseller != "RS-01" {
		t.Fatalf("kode_reseller tidak terisi: %+v", items[0].KodeReseller)
	}
	if got := env.pg.queryLog(); len(got) != 1 || !strings.Contains(got[0], "FROM inbox") {
		t.Fatalf("query tidak sesuai: %v", got)
	}
}

func TestInboxPGGetBerhasilTanpaHalamanBerikutnya(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: inboxRowValues(12), remaining: 1}, nil
	}

	items, hasNext, err := NewInboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, inboxFilterUji())
	if err != nil || hasNext || len(items) != 1 {
		t.Fatalf("items=%d hasNext=%v err=%v", len(items), hasNext, err)
	}
}

func TestInboxPGGetGagalSaatQuery(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) { return nil, errScripted }

	_, _, err := NewInboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, inboxFilterUji())
	if !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
}

func TestInboxPGGetGagalSaatScan(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: inboxRowValues(13), remaining: 1, scanErr: errScripted}, nil
	}

	_, _, err := NewInboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, inboxFilterUji())
	if !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
}

func TestInboxPGGetGagalSaatIterasi(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: inboxRowValues(14), remaining: 0, iterErr: errScripted}, nil
	}

	_, _, err := NewInboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, inboxFilterUji())
	if !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
}

func TestInboxPGGetBatasAtasKosongTetapLanjut(t *testing.T) {
	env := newFakeEnv(t)
	filter := inboxFilterUji()
	filter.EndDate = ptr(ujiWaktu())
	env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
		if strings.Contains(q, "tgl_entri <=") {
			return &pgRowFake{scanErr: pgx.ErrNoRows}
		}
		return &pgRowFake{values: []any{int64(999)}}
	}
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: inboxRowValues(15), remaining: 1}, nil
	}

	items, _, err := NewInboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, filter)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
}

func TestInboxPGGetGagalSaatMemotongBatasAtasDanBawah(t *testing.T) {
	for _, tc := range []struct {
		nama string
		atur func(*domain.InboxFilter)
		cari string
	}{
		{"cutEnd", func(f *domain.InboxFilter) { f.EndDate = ptr(ujiWaktu()) }, "tgl_entri <="},
		{"cutStart", func(f *domain.InboxFilter) { f.StartDate = ptr(ujiWaktu()) }, "tgl_entri <"},
	} {
		t.Run(tc.nama, func(t *testing.T) {
			env := newFakeEnv(t)
			filter := inboxFilterUji()
			tc.atur(&filter)
			env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
				if strings.Contains(q, tc.cari) {
					return &pgRowFake{scanErr: errScripted}
				}
				return &pgRowFake{values: []any{int64(1)}}
			}

			_, _, err := NewInboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, filter)
			if !errors.Is(err, errScripted) {
				t.Fatalf("err harus errScripted, dapat %v", err)
			}
		})
	}
}

func TestInboxPGInsertMemakaiKolomDanArgumenBenar(t *testing.T) {
	env := newFakeEnv(t)
	var jumlahArg int
	env.pg.execFn = func(q string, args []any) (pgconn.CommandTag, error) {
		jumlahArg = len(args)
		if !strings.Contains(q, "INSERT INTO inbox") {
			t.Errorf("SQL bukan insert inbox: %s", q)
		}
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}

	if err := NewInboxRepositories(env.reg).PG.Insert(context.Background(), tenantUji, sampleInbox()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if jumlahArg != 14 {
		t.Fatalf("harus 14 argumen, dapat %d", jumlahArg)
	}
}

func TestInboxPGInsertGagalDanTenantTidakTerdaftar(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.execFn = func(string, []any) (pgconn.CommandTag, error) { return pgconn.CommandTag{}, errScripted }
	repo := NewInboxRepositories(env.reg).PG

	if err := repo.Insert(context.Background(), tenantUji, sampleInbox()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Insert(context.Background(), "tenant-hilang", sampleInbox()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestInboxPGUpdateMengisiTglStatusHanyaSaatStatusDiisi(t *testing.T) {
	env := newFakeEnv(t)
	var argumen []any
	env.pg.execFn = func(q string, args []any) (pgconn.CommandTag, error) {
		argumen = args
		if !strings.Contains(q, "UPDATE inbox SET") || !strings.Contains(q, "WHERE kode = $15") {
			t.Errorf("SQL update tidak sesuai: %s", q)
		}
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	repo := NewInboxRepositories(env.reg).PG

	if err := repo.Update(context.Background(), tenantUji, 42, updateInboxRequestUji()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if len(argumen) != 15 {
		t.Fatalf("harus 15 argumen, dapat %d", len(argumen))
	}
	if !adaNilaiWaktu(argumen[0]) {
		t.Fatal("tgl_status harus terisi saat status diisi")
	}
	if argumen[len(argumen)-1] != int64(42) {
		t.Fatalf("kode harus argumen terakhir, dapat %v", argumen[len(argumen)-1])
	}

	if err := repo.Update(context.Background(), tenantUji, 43, dto.UpdateInboxRequest{}); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if adaNilaiWaktu(argumen[0]) {
		t.Fatal("tgl_status harus kosong saat status tidak diisi")
	}
}

func adaNilaiWaktu(v any) bool {
	p, ok := v.(*time.Time)
	return ok && p != nil
}

func TestInboxPGUpdateErrorDanTenantTidakTerdaftar(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.execFn = func(string, []any) (pgconn.CommandTag, error) { return pgconn.CommandTag{}, errScripted }
	repo := NewInboxRepositories(env.reg).PG

	if err := repo.Update(context.Background(), tenantUji, 1, updateInboxRequestUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Update(context.Background(), "tenant-hilang", 1, updateInboxRequestUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestInboxPGGetFilterLengkapDanTenantTidakTerdaftar(t *testing.T) {
	env := newFakeEnv(t)
	filter := inboxFilterUji()
	filter.StartDate = ptr(ujiWaktu())
	filter.EndDate = ptr(ujiWaktu())
	filter.LowerBound = 100
	filter.Cursor = 900
	filter.Terminal = ptr(7)
	filter.Reseller = ptr("RS-01")
	filter.Pengirim = ptr("Budi")
	filter.Tipe = ptr("hp")
	filter.Status = ptr(int16(1))
	filter.Pesan = "halo"
	env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
		switch {
		case strings.Contains(q, "tgl_entri <="):
			return &pgRowFake{values: []any{int64(500)}}
		case strings.Contains(q, "tgl_entri <"):
			return &pgRowFake{values: []any{int64(100)}}
		}
		return &pgRowFake{scanErr: errPGNotScripted}
	}
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: inboxRowValues(16), remaining: 1}, nil
	}
	repo := NewInboxRepositories(env.reg).PG

	if _, _, err := repo.Get(context.Background(), tenantUji, filter); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if got := env.pg.queryLog(); len(got) != 1 {
		t.Fatalf("harus 1 Query, dapat %d", len(got))
	}
	if _, _, err := repo.Get(context.Background(), "tenant-hilang", inboxFilterUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestInboxMSGetDenganBatasDanKursor(t *testing.T) {
	env := newFakeEnv(t)
	filter := inboxFilterUji()
	filter.LowerBound = 100
	filter.Cursor = 900
	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{columns: inboxKolom(), data: [][]driver.Value{msNilaiInbox(17)}}, nil
	}
	if _, _, err := NewInboxRepositories(env.reg).MS.Get(context.Background(), tenantUji, filter); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
}

func TestInboxPGLowerBoundDenganBatasAtasBerhasil(t *testing.T) {
	env := newFakeEnv(t)
	filter := inboxFilterUji()
	filter.EndDate = ptr(ujiWaktu())
	env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
		if strings.Contains(q, "OFFSET") {
			return &pgRowFake{values: []any{int64(321)}}
		}
		return &pgRowFake{values: []any{int64(500)}}
	}

	kode, err := NewInboxRepositories(env.reg).PG.LowerBound(context.Background(), tenantUji, filter)
	if err != nil || kode != 321 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}
}

func TestInboxPGLowerBoundMengembalikanKodeTerakhir(t *testing.T) {
	env := newFakeEnv(t)
	filter := inboxFilterUji()
	filter.StartDate = ptr(ujiWaktu())
	filter.LowerBound = 100
	filter.Cursor = 900
	env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
		if strings.Contains(q, "OFFSET") {
			return &pgRowFake{values: []any{int64(777)}}
		}
		return &pgRowFake{values: []any{int64(555)}}
	}

	kode, err := NewInboxRepositories(env.reg).PG.LowerBound(context.Background(), tenantUji, filter)
	if err != nil || kode != 777 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}
	if got := env.pg.rowLog(); len(got) != 2 {
		t.Fatalf("harus 2 QueryRow, dapat %d: %v", len(got), got)
	}
}

func TestInboxPGLowerBoundKososongDanError(t *testing.T) {
	for _, tc := range []struct {
		nama    string
		scanErr error
		harapan int64
	}{
		{"tanpa_baris", pgx.ErrNoRows, 0},
		{"error", errScripted, 0},
	} {
		t.Run(tc.nama, func(t *testing.T) {
			env := newFakeEnv(t)
			env.pg.queryRowFn = func(string, []any) pgx.Row { return &pgRowFake{scanErr: tc.scanErr} }

			kode, err := NewInboxRepositories(env.reg).PG.LowerBound(context.Background(), tenantUji, inboxFilterUji())
			if kode != tc.harapan {
				t.Fatalf("kode=%d", kode)
			}
			if tc.nama == "tanpa_baris" && err != nil {
				t.Fatalf("err harus nil, dapat %v", err)
			}
			if tc.nama == "error" && !errors.Is(err, errScripted) {
				t.Fatalf("err harus errScripted, dapat %v", err)
			}
		})
	}
}

func TestInboxPGLowerBoundGagalSaatMemotongDanTenantTidakTerdaftar(t *testing.T) {
	for _, tc := range []struct {
		nama string
		atur func(*domain.InboxFilter)
		cari string
	}{
		{"cutEnd", func(f *domain.InboxFilter) { f.EndDate = ptr(ujiWaktu()) }, "tgl_entri <="},
		{"cutStart", func(f *domain.InboxFilter) { f.StartDate = ptr(ujiWaktu()) }, "tgl_entri <"},
	} {
		t.Run(tc.nama, func(t *testing.T) {
			env := newFakeEnv(t)
			filter := inboxFilterUji()
			tc.atur(&filter)
			env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
				if strings.Contains(q, tc.cari) {
					return &pgRowFake{scanErr: errScripted}
				}
				return &pgRowFake{values: []any{int64(1)}}
			}
			if _, err := NewInboxRepositories(env.reg).PG.LowerBound(context.Background(), tenantUji, filter); !errors.Is(err, errScripted) {
				t.Fatalf("err harus errScripted, dapat %v", err)
			}
		})
	}

	env := newFakeEnv(t)
	if _, err := NewInboxRepositories(env.reg).PG.LowerBound(context.Background(), "tenant-hilang", inboxFilterUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

// ------------------------------------------------------------- Inbox: MSSQL

func TestInboxMSGetBerhasilDenganHalamanBerikutnya(t *testing.T) {
	env := newFakeEnv(t)
	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{
			columns: inboxKolom(),
			data:    [][]driver.Value{msNilaiInbox(21), msNilaiInbox(22)},
		}, nil
	}

	items, hasNext, err := NewInboxRepositories(env.reg).MS.Get(context.Background(), tenantUji, inboxFilterUji())
	if err != nil || !hasNext || len(items) != 1 || items[0].Kode != 21 {
		t.Fatalf("items=%d hasNext=%v err=%v", len(items), hasNext, err)
	}
	if got := env.ms.queryLog(); len(got) != 1 || !strings.Contains(got[0], "FROM inbox") {
		t.Fatalf("query tidak sesuai: %v", got)
	}
}

func TestInboxMSGetBerhasilTanpaHalamanBerikutnyaDanGagal(t *testing.T) {
	env := newFakeEnv(t)
	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{columns: inboxKolom(), data: [][]driver.Value{msNilaiInbox(23)}}, nil
	}
	repo := NewInboxRepositories(env.reg).MS
	items, hasNext, err := repo.Get(context.Background(), tenantUji, inboxFilterUji())
	if err != nil || hasNext || len(items) != 1 {
		t.Fatalf("items=%d hasNext=%v err=%v", len(items), hasNext, err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) { return nil, errScripted }
	if _, _, err := repo.Get(context.Background(), tenantUji, inboxFilterUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	if _, _, err := repo.Get(context.Background(), "tenant-hilang", inboxFilterUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestInboxMSGetGagalSaatScanDanIterasi(t *testing.T) {
	env := newFakeEnv(t)
	repo := NewInboxRepositories(env.reg).MS

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		baris := msNilaiInbox(24)
		baris[0] = tipeTidakCocok{}
		return &msRowsFake{columns: inboxKolom(), data: [][]driver.Value{baris}}, nil
	}
	if _, _, err := repo.Get(context.Background(), tenantUji, inboxFilterUji()); err == nil {
		t.Fatal("harus gagal saat scan tipe tidak cocok")
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{
			columns: inboxKolom(),
			data:    [][]driver.Value{msNilaiInbox(25), msNilaiInbox(26)},
			failAt:  1,
			failErr: errStreamBroken,
		}, nil
	}
	if _, _, err := repo.Get(context.Background(), tenantUji, inboxFilterUji()); !errors.Is(err, errStreamBroken) {
		t.Fatalf("err harus errStreamBroken, dapat %v", err)
	}
}

func TestInboxMSInsertDanUpdateMengirimArgumenBenar(t *testing.T) {
	env := newFakeEnv(t)
	var namaArgumen []string
	env.ms.execFn = func(_ string, args []driver.NamedValue) (driver.Result, error) {
		namaArgumen = namaArgumen[:0]
		for _, a := range args {
			namaArgumen = append(namaArgumen, a.Name)
		}
		return msResultFake{affected: 1}, nil
	}
	repo := NewInboxRepositories(env.reg).MS

	if err := repo.Insert(context.Background(), tenantUji, sampleInbox()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if len(namaArgumen) != 14 || namaArgumen[0] != "tgl_entri" {
		t.Fatalf("argumen insert tidak sesuai: %v", namaArgumen)
	}

	if err := repo.Update(context.Background(), tenantUji, 42, updateInboxRequestUji()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if len(namaArgumen) != 15 || namaArgumen[14] != "kode" {
		t.Fatalf("argumen update tidak sesuai: %v", namaArgumen)
	}

	execs := env.ms.execLog()
	if len(execs) != 2 || !strings.Contains(execs[0], "INSERT INTO inbox") || !strings.Contains(execs[1], "WHERE kode = @kode") {
		t.Fatalf("SQL tidak sesuai: %v", execs)
	}
}

func TestInboxMSInsertUpdateErrorDanTenantTidakTerdaftar(t *testing.T) {
	env := newFakeEnv(t)
	env.ms.execFn = func(string, []driver.NamedValue) (driver.Result, error) { return nil, errScripted }
	repo := NewInboxRepositories(env.reg).MS

	if err := repo.Insert(context.Background(), tenantUji, sampleInbox()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Update(context.Background(), tenantUji, 1, updateInboxRequestUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Insert(context.Background(), "tenant-hilang", sampleInbox()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
	if err := repo.Update(context.Background(), "tenant-hilang", 1, updateInboxRequestUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestInboxMSUpdateTglStatusMengikutiStatus(t *testing.T) {
	env := newFakeEnv(t)
	var tglStatusSet bool
	env.ms.execFn = func(_ string, args []driver.NamedValue) (driver.Result, error) {
		for _, a := range args {
			if a.Name == "tgl_status" {
				if p, ok := a.Value.(*time.Time); ok && p != nil {
					tglStatusSet = true
				}
			}
		}
		return msResultFake{affected: 1}, nil
	}
	repo := NewInboxRepositories(env.reg).MS

	if err := repo.Update(context.Background(), tenantUji, 1, dto.UpdateInboxRequest{}); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if tglStatusSet {
		t.Fatal("tgl_status harus kosong saat status tidak diisi")
	}

	if err := repo.Update(context.Background(), tenantUji, 1, updateInboxRequestUji()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if !tglStatusSet {
		t.Fatal("tgl_status harus terisi saat status diisi")
	}
}

func TestInboxMSLowerBoundBerhasilKosongDanError(t *testing.T) {
	env := newFakeEnv(t)
	filter := inboxFilterUji()
	filter.StartDate = ptr(ujiWaktu())
	env.ms.queryFn = func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "TOP (1)") {
			return &msRowsFake{columns: []string{"kode"}, data: [][]driver.Value{{int64(555)}}}, nil
		}
		return &msRowsFake{columns: []string{"kode"}, data: [][]driver.Value{{int64(888)}}}, nil
	}
	kode, err := NewInboxRepositories(env.reg).MS.LowerBound(context.Background(), tenantUji, filter)
	if err != nil || kode != 888 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{columns: []string{"kode"}}, nil
	}
	kode, err = NewInboxRepositories(env.reg).MS.LowerBound(context.Background(), tenantUji, filter)
	if err != nil || kode != 0 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) { return nil, errScripted }
	if _, err = NewInboxRepositories(env.reg).MS.LowerBound(context.Background(), tenantUji, filter); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	if _, err = NewInboxRepositories(env.reg).MS.LowerBound(context.Background(), "tenant-hilang", filter); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestInboxMSLowerBoundGagalSaatMemotongBatasAtas(t *testing.T) {
	env := newFakeEnv(t)
	filter := inboxFilterUji()
	filter.EndDate = ptr(ujiWaktu())
	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) { return nil, errScripted }

	if _, err := NewInboxRepositories(env.reg).MS.LowerBound(context.Background(), tenantUji, filter); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
}

func TestInboxMSLowerBoundBatasAtasKosongTetapLanjut(t *testing.T) {
	env := newFakeEnv(t)
	filter := inboxFilterUji()
	filter.EndDate = ptr(ujiWaktu())
	env.ms.queryFn = func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "TOP (1)") {
			return &msRowsFake{columns: []string{"kode"}}, nil
		}
		return &msRowsFake{columns: []string{"kode"}, data: [][]driver.Value{{int64(64)}}}, nil
	}

	kode, err := NewInboxRepositories(env.reg).MS.LowerBound(context.Background(), tenantUji, filter)
	if err != nil || kode != 64 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}
}

// ------------------------------------------------------------ Outbox: PG

func TestOutboxPGGetBerhasilDenganDanTanpaHalamanBerikutnya(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: outboxRowValues(31), remaining: 2}, nil
	}
	repo := NewOutboxRepositories(env.reg).PG

	items, hasNext, err := repo.Get(context.Background(), tenantUji, outboxFilterUji())
	if err != nil || !hasNext || len(items) != 1 || items[0].Kode != 31 {
		t.Fatalf("items=%d hasNext=%v err=%v", len(items), hasNext, err)
	}
	if items[0].Penerima != "0812999" {
		t.Fatalf("penerima tidak sesuai: %s", items[0].Penerima)
	}

	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: outboxRowValues(32), remaining: 1}, nil
	}
	items, hasNext, err = repo.Get(context.Background(), tenantUji, outboxFilterUji())
	if err != nil || hasNext || len(items) != 1 {
		t.Fatalf("items=%d hasNext=%v err=%v", len(items), hasNext, err)
	}
}

func TestOutboxPGGetDenganBatasKursorDanFilterLengkap(t *testing.T) {
	env := newFakeEnv(t)
	filter := outboxFilterUji()
	filter.StartDate = ptr(ujiWaktu())
	filter.LowerBound = 100
	filter.Cursor = 900
	filter.Reseller = ptr("RS-01")
	filter.Penerima = ptr("0812999")
	filter.Tipe = ptr("hp")
	filter.Status = ptr(int16(1))
	filter.Pesan = "halo"
	env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
		if strings.Contains(q, "tgl_entri <") {
			return &pgRowFake{values: []any{int64(100)}}
		}
		return &pgRowFake{scanErr: errPGNotScripted}
	}
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: outboxRowValues(36), remaining: 1}, nil
	}
	if _, _, err := NewOutboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, filter); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
}

func TestOutboxMSGetDenganBatasDanKursor(t *testing.T) {
	env := newFakeEnv(t)
	filter := outboxFilterUji()
	filter.LowerBound = 100
	filter.Cursor = 900
	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{columns: outboxKolom(), data: [][]driver.Value{msNilaiOutbox(37)}}, nil
	}
	if _, _, err := NewOutboxRepositories(env.reg).MS.Get(context.Background(), tenantUji, filter); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
}

func TestOutboxPGGetSemuaCabangGagal(t *testing.T) {
	env := newFakeEnv(t)
	repo := NewOutboxRepositories(env.reg).PG

	env.pg.queryFn = func(string, []any) (pgx.Rows, error) { return nil, errScripted }
	if _, _, err := repo.Get(context.Background(), tenantUji, outboxFilterUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: outboxRowValues(33), remaining: 1, scanErr: errScripted}, nil
	}
	if _, _, err := repo.Get(context.Background(), tenantUji, outboxFilterUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: outboxRowValues(34), iterErr: errScripted}, nil
	}
	if _, _, err := repo.Get(context.Background(), tenantUji, outboxFilterUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	if _, _, err := repo.Get(context.Background(), "tenant-hilang", outboxFilterUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestOutboxPGGetMemotongBatasDanSeluruhnyaGagal(t *testing.T) {
	env := newFakeEnv(t)
	repo := NewOutboxRepositories(env.reg).PG

	filter := outboxFilterUji()
	filter.EndDate = ptr(ujiWaktu())
	filter.StartDate = ptr(ujiWaktu())
	env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
		if strings.Contains(q, "tgl_entri <") {
			return &pgRowFake{values: []any{int64(10)}}
		}
		return &pgRowFake{values: []any{int64(20)}}
	}
	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: outboxRowValues(35), remaining: 1}, nil
	}
	if _, _, err := repo.Get(context.Background(), tenantUji, filter); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}

	for _, tc := range []struct {
		nama string
		cari string
	}{
		{"cutEnd", "tgl_entri <="},
		{"cutStart", "tgl_entri <"},
	} {
		t.Run(tc.nama, func(t *testing.T) {
			env := newFakeEnv(t)
			filter := outboxFilterUji()
			if tc.nama == "cutEnd" {
				filter.EndDate = ptr(ujiWaktu())
			} else {
				filter.StartDate = ptr(ujiWaktu())
			}
			env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
				if strings.Contains(q, tc.cari) {
					return &pgRowFake{scanErr: errScripted}
				}
				return &pgRowFake{values: []any{int64(1)}}
			}
			if _, _, err := NewOutboxRepositories(env.reg).PG.Get(context.Background(), tenantUji, filter); !errors.Is(err, errScripted) {
				t.Fatalf("err harus errScripted, dapat %v", err)
			}
		})
	}
}

func TestOutboxPGInsertMemakaiKolomDanArgumenBenar(t *testing.T) {
	env := newFakeEnv(t)
	var jumlahArg int
	env.pg.execFn = func(q string, args []any) (pgconn.CommandTag, error) {
		jumlahArg = len(args)
		if !strings.Contains(q, "INSERT INTO outbox") {
			t.Errorf("SQL bukan insert outbox: %s", q)
		}
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}

	if err := NewOutboxRepositories(env.reg).PG.Insert(context.Background(), tenantUji, sampleOutbox()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if jumlahArg != 16 {
		t.Fatalf("harus 16 argumen, dapat %d", jumlahArg)
	}
}

func TestOutboxPGInsertUpdateErrorDanTenantTidakTerdaftar(t *testing.T) {
	env := newFakeEnv(t)
	env.pg.execFn = func(string, []any) (pgconn.CommandTag, error) { return pgconn.CommandTag{}, errScripted }
	repo := NewOutboxRepositories(env.reg).PG

	if err := repo.Insert(context.Background(), tenantUji, sampleOutbox()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Update(context.Background(), tenantUji, 1, updateOutboxRequestUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Insert(context.Background(), "tenant-hilang", sampleOutbox()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
	if err := repo.Update(context.Background(), "tenant-hilang", 1, updateOutboxRequestUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestOutboxPGUpdateMemakaiTglStatusDanKodeTerakhir(t *testing.T) {
	env := newFakeEnv(t)
	var argumen []any
	env.pg.execFn = func(q string, args []any) (pgconn.CommandTag, error) {
		argumen = args
		if !strings.Contains(q, "WHERE kode = $17") {
			t.Errorf("SQL update tidak sesuai: %s", q)
		}
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	repo := NewOutboxRepositories(env.reg).PG

	if err := repo.Update(context.Background(), tenantUji, 51, updateOutboxRequestUji()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if len(argumen) != 17 || !adaNilaiWaktu(argumen[0]) || argumen[16] != int64(51) {
		t.Fatalf("argumen update tidak sesuai: %v", argumen)
	}

	if err := repo.Update(context.Background(), tenantUji, 52, dto.UpdateOutboxRequest{}); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if adaNilaiWaktu(argumen[0]) {
		t.Fatal("tgl_status harus kosong saat status tidak diisi")
	}
}

func TestOutboxMSLowerBoundGagalSaatMemotongBatasAtas(t *testing.T) {
	env := newFakeEnv(t)
	filter := outboxFilterUji()
	filter.EndDate = ptr(ujiWaktu())
	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) { return nil, errScripted }

	if _, err := NewOutboxRepositories(env.reg).MS.LowerBound(context.Background(), tenantUji, filter); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
}

func TestOutboxPGLowerBoundSemuaCabang(t *testing.T) {
	env := newFakeEnv(t)
	filter := outboxFilterUji()
	filter.StartDate = ptr(ujiWaktu())
	filter.EndDate = ptr(ujiWaktu())
	filter.LowerBound = 5
	filter.Cursor = 700
	env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
		switch {
		case strings.Contains(q, "OFFSET"):
			return &pgRowFake{values: []any{int64(654)}}
		case strings.Contains(q, "tgl_entri <="):
			return &pgRowFake{values: []any{int64(222)}}
		default:
			return &pgRowFake{values: []any{int64(111)}}
		}
	}
	repo := NewOutboxRepositories(env.reg).PG

	kode, err := repo.LowerBound(context.Background(), tenantUji, filter)
	if err != nil || kode != 654 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}

	env.pg.queryRowFn = func(string, []any) pgx.Row { return &pgRowFake{scanErr: pgx.ErrNoRows} }
	if kode, err = repo.LowerBound(context.Background(), tenantUji, filter); err != nil || kode != 0 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}

	env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
		if strings.Contains(q, "OFFSET") {
			return &pgRowFake{scanErr: errScripted}
		}
		return &pgRowFake{values: []any{int64(111)}}
	}
	if _, err = repo.LowerBound(context.Background(), tenantUji, filter); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	if _, err = repo.LowerBound(context.Background(), "tenant-hilang", filter); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestOutboxPGLowerBoundGagalSaatMemotong(t *testing.T) {
	for _, tc := range []struct {
		nama string
		cari string
	}{
		{"cutEnd", "tgl_entri <="},
		{"cutStart", "tgl_entri <"},
	} {
		t.Run(tc.nama, func(t *testing.T) {
			env := newFakeEnv(t)
			filter := outboxFilterUji()
			if tc.nama == "cutEnd" {
				filter.EndDate = ptr(ujiWaktu())
			} else {
				filter.StartDate = ptr(ujiWaktu())
			}
			env.pg.queryRowFn = func(q string, _ []any) pgx.Row {
				if strings.Contains(q, tc.cari) {
					return &pgRowFake{scanErr: errScripted}
				}
				return &pgRowFake{values: []any{int64(1)}}
			}
			if _, err := NewOutboxRepositories(env.reg).PG.LowerBound(context.Background(), tenantUji, filter); !errors.Is(err, errScripted) {
				t.Fatalf("err harus errScripted, dapat %v", err)
			}
		})
	}
}

// ------------------------------------------------------------ Outbox: MSSQL

func TestOutboxMSGetBerhasilDenganDanTanpaHalamanBerikutnya(t *testing.T) {
	env := newFakeEnv(t)
	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{columns: outboxKolom(), data: [][]driver.Value{msNilaiOutbox(41), msNilaiOutbox(42)}}, nil
	}
	repo := NewOutboxRepositories(env.reg).MS

	items, hasNext, err := repo.Get(context.Background(), tenantUji, outboxFilterUji())
	if err != nil || !hasNext || len(items) != 1 || items[0].Kode != 41 {
		t.Fatalf("items=%d hasNext=%v err=%v", len(items), hasNext, err)
	}
	if items[0].KodeReseller == nil || *items[0].KodeReseller != "RS-01" {
		t.Fatalf("kode_reseller tidak sesuai: %+v", items[0].KodeReseller)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{columns: outboxKolom(), data: [][]driver.Value{msNilaiOutbox(43)}}, nil
	}
	items, hasNext, err = repo.Get(context.Background(), tenantUji, outboxFilterUji())
	if err != nil || hasNext || len(items) != 1 {
		t.Fatalf("items=%d hasNext=%v err=%v", len(items), hasNext, err)
	}
}

func TestOutboxMSGetSemuaCabangGagal(t *testing.T) {
	env := newFakeEnv(t)
	repo := NewOutboxRepositories(env.reg).MS

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) { return nil, errScripted }
	if _, _, err := repo.Get(context.Background(), tenantUji, outboxFilterUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		baris := msNilaiOutbox(44)
		baris[0] = tipeTidakCocok{}
		return &msRowsFake{columns: outboxKolom(), data: [][]driver.Value{baris}}, nil
	}
	if _, _, err := repo.Get(context.Background(), tenantUji, outboxFilterUji()); err == nil {
		t.Fatal("harus gagal saat scan tipe tidak cocok")
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{
			columns: outboxKolom(),
			data:    [][]driver.Value{msNilaiOutbox(45), msNilaiOutbox(46)},
			failAt:  1,
			failErr: errStreamBroken,
		}, nil
	}
	if _, _, err := repo.Get(context.Background(), tenantUji, outboxFilterUji()); !errors.Is(err, errStreamBroken) {
		t.Fatalf("err harus errStreamBroken, dapat %v", err)
	}

	if _, _, err := repo.Get(context.Background(), "tenant-hilang", outboxFilterUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestOutboxMSInsertUpdateMengirimArgumenBenar(t *testing.T) {
	env := newFakeEnv(t)
	var namaArgumen []string
	env.ms.execFn = func(_ string, args []driver.NamedValue) (driver.Result, error) {
		namaArgumen = namaArgumen[:0]
		for _, a := range args {
			namaArgumen = append(namaArgumen, a.Name)
		}
		return msResultFake{affected: 1}, nil
	}
	repo := NewOutboxRepositories(env.reg).MS

	if err := repo.Insert(context.Background(), tenantUji, sampleOutbox()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if len(namaArgumen) != 16 || namaArgumen[0] != "tgl_entri" {
		t.Fatalf("argumen insert tidak sesuai: %v", namaArgumen)
	}

	if err := repo.Update(context.Background(), tenantUji, 61, updateOutboxRequestUji()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if len(namaArgumen) != 17 || namaArgumen[16] != "kode" {
		t.Fatalf("argumen update tidak sesuai: %v", namaArgumen)
	}

	execs := env.ms.execLog()
	if len(execs) != 2 || !strings.Contains(execs[0], "INSERT INTO outbox") || !strings.Contains(execs[1], "WHERE kode = @kode") {
		t.Fatalf("SQL tidak sesuai: %v", execs)
	}
}

func TestOutboxMSUpdateTglStatusDanSeluruhCabangGagal(t *testing.T) {
	env := newFakeEnv(t)
	var tglStatusSet bool
	env.ms.execFn = func(_ string, args []driver.NamedValue) (driver.Result, error) {
		for _, a := range args {
			if a.Name == "tgl_status" {
				if p, ok := a.Value.(*time.Time); ok && p != nil {
					tglStatusSet = true
				}
			}
		}
		return msResultFake{affected: 1}, nil
	}
	repo := NewOutboxRepositories(env.reg).MS

	if err := repo.Update(context.Background(), tenantUji, 1, dto.UpdateOutboxRequest{}); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if tglStatusSet {
		t.Fatal("tgl_status harus kosong saat status tidak diisi")
	}

	if err := repo.Update(context.Background(), tenantUji, 1, updateOutboxRequestUji()); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if !tglStatusSet {
		t.Fatal("tgl_status harus terisi saat status diisi")
	}

	env.ms.execFn = func(string, []driver.NamedValue) (driver.Result, error) { return nil, errScripted }
	if err := repo.Insert(context.Background(), tenantUji, sampleOutbox()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Update(context.Background(), tenantUji, 1, updateOutboxRequestUji()); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Insert(context.Background(), "tenant-hilang", sampleOutbox()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
	if err := repo.Update(context.Background(), "tenant-hilang", 1, updateOutboxRequestUji()); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestOutboxMSLowerBoundSemuaCabang(t *testing.T) {
	env := newFakeEnv(t)
	filter := outboxFilterUji()
	filter.EndDate = ptr(ujiWaktu())
	env.ms.queryFn = func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "TOP (1)") {
			return &msRowsFake{columns: []string{"kode"}, data: [][]driver.Value{{int64(321)}}}, nil
		}
		return &msRowsFake{columns: []string{"kode"}, data: [][]driver.Value{{int64(999)}}}, nil
	}
	repo := NewOutboxRepositories(env.reg).MS

	kode, err := repo.LowerBound(context.Background(), tenantUji, filter)
	if err != nil || kode != 999 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}

	env.ms.queryFn = func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "TOP (1)") {
			return &msRowsFake{columns: []string{"kode"}}, nil
		}
		return &msRowsFake{columns: []string{"kode"}}, nil
	}
	if kode, err = repo.LowerBound(context.Background(), tenantUji, filter); err != nil || kode != 0 {
		t.Fatalf("kode=%d err=%v", kode, err)
	}

	env.ms.queryFn = func(q string, _ []driver.NamedValue) (driver.Rows, error) {
		if strings.Contains(q, "TOP (1)") {
			return &msRowsFake{columns: []string{"kode"}, data: [][]driver.Value{{int64(321)}}}, nil
		}
		return nil, errScripted
	}
	if _, err = repo.LowerBound(context.Background(), tenantUji, filter); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) { return nil, errScripted }
	if _, err = repo.LowerBound(context.Background(), "tenant-hilang", filter); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

// ------------------------------------------------------------------- User

func TestUserPGCariDanSeluruhCabangGagal(t *testing.T) {
	env := newFakeEnv(t)
	repo := NewUserRepositories(env.reg).PG

	env.pg.queryRowFn = func(string, []any) pgx.Row {
		return &pgRowFake{values: []any{1, "admin", "rahasia", "all"}}
	}
	u, err := repo.GetByUsername(context.Background(), tenantUji, "admin")
	if err != nil || u.Username != "admin" || u.UserID != 1 || u.Rules != "all" {
		t.Fatalf("user=%+v err=%v", u, err)
	}

	env.pg.queryRowFn = func(string, []any) pgx.Row { return &pgRowFake{scanErr: pgx.ErrNoRows} }
	if _, err = repo.GetByUsername(context.Background(), tenantUji, "hilang"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("err harus ErrNoRows, dapat %v", err)
	}

	env.pg.queryRowFn = func(string, []any) pgx.Row { return &pgRowFake{scanErr: errScripted} }
	if _, err = repo.GetByUsername(context.Background(), tenantUji, "admin"); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	if _, err = repo.GetByUsername(context.Background(), "tenant-hilang", "admin"); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestUserPGInsertUpdateBerhasilDanGagal(t *testing.T) {
	env := newFakeEnv(t)
	var jumlahArg int
	env.pg.execFn = func(q string, args []any) (pgconn.CommandTag, error) {
		jumlahArg = len(args)
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	repo := NewUserRepositories(env.reg).PG

	if err := repo.Insert(context.Background(), tenantUji, domain.User{Username: "u", Password: "p", Rules: "r"}); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if jumlahArg != 3 {
		t.Fatalf("harus 3 argumen, dapat %d", jumlahArg)
	}

	if err := repo.Update(context.Background(), tenantUji, "u", ptr("p2"), ptr("r2")); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if jumlahArg != 3 {
		t.Fatalf("harus 3 argumen, dapat %d", jumlahArg)
	}

	env.pg.execFn = func(string, []any) (pgconn.CommandTag, error) { return pgconn.CommandTag{}, errScripted }
	if err := repo.Insert(context.Background(), tenantUji, domain.User{}); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Update(context.Background(), tenantUji, "u", nil, nil); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Insert(context.Background(), "tenant-hilang", domain.User{}); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
	if err := repo.Update(context.Background(), "tenant-hilang", "u", nil, nil); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestUserMSCariDanSeluruhCabangGagal(t *testing.T) {
	env := newFakeEnv(t)
	repo := NewUserRepositories(env.reg).MS

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{
			columns: []string{"userid", "username", "pass", "rules"},
			data:    [][]driver.Value{{int64(1), "admin", "rahasia", "all"}},
		}, nil
	}
	u, err := repo.GetByUsername(context.Background(), tenantUji, "admin")
	if err != nil || u.Password != "rahasia" {
		t.Fatalf("user=%+v err=%v", u, err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{columns: []string{"userid", "username", "pass", "rules"}}, nil
	}
	if _, err = repo.GetByUsername(context.Background(), tenantUji, "hilang"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("err harus sql.ErrNoRows, dapat %v", err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) { return nil, errScripted }
	if _, err = repo.GetByUsername(context.Background(), tenantUji, "admin"); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if _, err = repo.GetByUsername(context.Background(), "tenant-hilang", "admin"); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestUserMSInsertUpdateBerhasilDanGagal(t *testing.T) {
	env := newFakeEnv(t)
	var namaArgumen []string
	env.ms.execFn = func(_ string, args []driver.NamedValue) (driver.Result, error) {
		namaArgumen = namaArgumen[:0]
		for _, a := range args {
			namaArgumen = append(namaArgumen, a.Name)
		}
		return msResultFake{affected: 1}, nil
	}
	repo := NewUserRepositories(env.reg).MS

	if err := repo.Insert(context.Background(), tenantUji, domain.User{Username: "u"}); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if strings.Join(namaArgumen, ",") != "username,password,rules" {
		t.Fatalf("argumen insert tidak sesuai: %v", namaArgumen)
	}

	if err := repo.Update(context.Background(), tenantUji, "u", ptr("p2"), ptr("r2")); err != nil {
		t.Fatalf("err unexpected: %v", err)
	}
	if strings.Join(namaArgumen, ",") != "password,rules,username" {
		t.Fatalf("argumen update tidak sesuai: %v", namaArgumen)
	}

	env.ms.execFn = func(string, []driver.NamedValue) (driver.Result, error) { return nil, errScripted }
	if err := repo.Insert(context.Background(), tenantUji, domain.User{}); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Update(context.Background(), tenantUji, "u", nil, nil); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}
	if err := repo.Insert(context.Background(), "tenant-hilang", domain.User{}); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
	if err := repo.Update(context.Background(), "tenant-hilang", "u", nil, nil); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

// --------------------------------------------------------------- Reseller

func TestResellerDropdownPGBerhasilDanSeluruhCabangGagal(t *testing.T) {
	env := newFakeEnv(t)
	repo := NewResellerRepositories(env.reg).PG

	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: resellerRowValues("RS-01", "Reseller Satu"), remaining: 2}, nil
	}
	hasil, err := repo.ListForDropdown(context.Background(), tenantUji)
	if err != nil || len(hasil) != 2 || hasil[0].Kode != "RS-01" {
		t.Fatalf("hasil=%+v err=%v", hasil, err)
	}

	env.pg.queryFn = func(string, []any) (pgx.Rows, error) { return nil, errScripted }
	if _, err = repo.ListForDropdown(context.Background(), tenantUji); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: resellerRowValues("RS-01", "Reseller Satu"), remaining: 1, scanErr: errScripted}, nil
	}
	if _, err = repo.ListForDropdown(context.Background(), tenantUji); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	env.pg.queryFn = func(string, []any) (pgx.Rows, error) {
		return &pgRowsFake{values: resellerRowValues("RS-01", "Reseller Satu"), iterErr: errScripted}, nil
	}
	if _, err = repo.ListForDropdown(context.Background(), tenantUji); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	if _, err = repo.ListForDropdown(context.Background(), "tenant-hilang"); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

func TestResellerDropdownMSBerhasilDanSeluruhCabangGagal(t *testing.T) {
	env := newFakeEnv(t)
	repo := NewResellerRepositories(env.reg).MS

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{
			columns: []string{"kode", "nama"},
			data: [][]driver.Value{
				{"RS-01", "Reseller Satu"},
				{"RS-02", "Reseller Dua"},
			},
		}, nil
	}
	hasil, err := repo.ListForDropdown(context.Background(), tenantUji)
	if err != nil || len(hasil) != 2 || hasil[1].Nama != "Reseller Dua" {
		t.Fatalf("hasil=%+v err=%v", hasil, err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) { return nil, errScripted }
	if _, err = repo.ListForDropdown(context.Background(), tenantUji); !errors.Is(err, errScripted) {
		t.Fatalf("err harus errScripted, dapat %v", err)
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{
			columns: []string{"kode", "nama"},
			data:    [][]driver.Value{{tipeTidakCocok{}, "Reseller Satu"}},
		}, nil
	}
	if _, err = repo.ListForDropdown(context.Background(), tenantUji); err == nil {
		t.Fatal("harus gagal saat scan tipe tidak cocok")
	}

	env.ms.queryFn = func(string, []driver.NamedValue) (driver.Rows, error) {
		return &msRowsFake{
			columns: []string{"kode", "nama"},
			data:    [][]driver.Value{{"RS-01", "Satu"}, {"RS-02", "Dua"}},
			failAt:  1,
			failErr: errStreamBroken,
		}, nil
	}
	if _, err = repo.ListForDropdown(context.Background(), tenantUji); !errors.Is(err, errStreamBroken) {
		t.Fatalf("err harus errStreamBroken, dapat %v", err)
	}

	if _, err = repo.ListForDropdown(context.Background(), "tenant-hilang"); !errors.Is(err, database.ErrTenantNotFound) {
		t.Fatalf("err harus ErrTenantNotFound, dapat %v", err)
	}
}

// ------------------------------------------------------------------ helper

func inboxKolom() []string {
	return []string{"kode", "tgl_entri", "pengirim", "kode_reseller", "pesan", "status", "tgl_status", "kode_terminal", "service_center"}
}

func outboxKolom() []string {
	return []string{"kode", "tgl_entri", "penerima", "kode_reseller", "pesan", "status", "tgl_status"}
}

func msNilaiInbox(kode int64) []driver.Value {
	return []driver.Value{
		kode, ujiWaktu(), "0812345", "RS-01", "halo", int16(1), nil, int64(7), "SC-1",
	}
}

func msNilaiOutbox(kode int64) []driver.Value {
	return []driver.Value{kode, ujiWaktu(), "0812999", "RS-01", "hai", int16(0), nil}
}
