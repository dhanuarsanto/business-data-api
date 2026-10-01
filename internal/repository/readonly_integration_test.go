//go:build integration

package repository_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/internal/repository"
	"go.internal/business-data-api/internal/usecase"
	db "go.internal/business-data-api/pkg/database"
)

var (
	repoOnce sync.Once
	initErr  error

	pgIn, msIn     domain.InboxRepository
	pgOut, msOut   domain.OutboxRepository
	pgUser, msUser domain.UserRepository

	rawPG *pgxpool.Pool
	rawMS *sql.DB
	dbReg *db.DBRegistry
)

func initRepos() {
	repoOnce.Do(func() {
		cfg := config.LoadConfig()
		ctx := context.Background()

		pg, err := db.NewPostgresPool(ctx, cfg.PostgresMaxtopURL, db.PostgresPoolOptions{})
		if err != nil {
			initErr = err
			return
		}
		ms, err := db.NewMSSQLDB(ctx, cfg.MSSQLMaxtopURL, db.MSSQLPoolOptions{})
		if err != nil {
			initErr = err
			return
		}

		reg := db.NewDBRegistry()
		reg.Register("maxtop", pg, ms)

		rawPG = pg
		rawMS = ms
		dbReg = reg

		ir := repository.NewInboxRepositories(reg)
		orr := repository.NewOutboxRepositories(reg)
		ur := repository.NewUserRepositories(reg)
		pgIn, msIn = ir.PG, ir.MS
		pgOut, msOut = orr.PG, orr.MS
		pgUser, msUser = ur.PG, ur.MS
	})
}

func newRegistry(t *testing.T) *db.DBRegistry {
	t.Helper()
	ready(t)
	return dbReg
}

func ready(t *testing.T) {
	if os.Getenv("INTEGRATION_DB") != "1" {
		t.Skip("INTEGRATION_DB != 1, lewati test DB")
	}
	initRepos()
	if initErr != nil {
		t.Skipf("koneksi DB tak tersedia: %v", initErr)
	}
}

func pInt(v int) *int       { return &v }
func pInt16(v int16) *int16 { return &v }

func TestBisectionMatchesLegacy(t *testing.T) {
	ready(t)
	ctx := context.Background()
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	check := func(table string, get func() ([]int64, error)) {
		gotCfg, err := get()
		if err != nil {
			t.Fatalf("%s: get gagal: %v", table, err)
		}

		rows, err := rawPG.Query(ctx, `SELECT kode FROM pandora_dw.staging.`+table+` WHERE tgl_entri <= $1 ORDER BY kode DESC LIMIT 5`, end)
		if err != nil {
			t.Fatalf("%s: query legacy gagal: %v", table, err)
		}
		defer rows.Close()
		var want []int64
		for rows.Next() {
			var k int64
			if err := rows.Scan(&k); err != nil {
				t.Fatalf("%s: scan: %v", table, err)
			}
			want = append(want, k)
		}

		if len(gotCfg) != len(want) {
			t.Fatalf("%s: jumlah baris beda: bisection=%d legacy=%d", table, len(gotCfg), len(want))
		}
		for i := range want {
			if gotCfg[i] != want[i] {
				t.Fatalf("%s: kode ke-%d beda: bisection=%d legacy=%d", table, i, gotCfg[i], want[i])
			}
		}
	}

	check("inbox", func() ([]int64, error) {
		data, _, err := pgIn.Get(ctx, "maxtop", domain.InboxFilter{PageSize: 5, EndDate: &end})
		if err != nil {
			return nil, err
		}
		ks := make([]int64, len(data))
		for i, d := range data {
			ks[i] = d.Kode
		}
		return ks, nil
	})

	check("outbox", func() ([]int64, error) {
		data, _, err := pgOut.Get(ctx, "maxtop", domain.OutboxFilter{PageSize: 5, EndDate: &end})
		if err != nil {
			return nil, err
		}
		ks := make([]int64, len(data))
		for i, d := range data {
			ks[i] = d.Kode
		}
		return ks, nil
	})
}

func TestPaginationWithFilters(t *testing.T) {
	ready(t)
	ctx := context.Background()
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	status := int16(20)
	pesan := "trx"

	legacy := func(table string, limit int) ([]int64, error) {
		rows, err := rawPG.Query(ctx, `SELECT kode FROM pandora_dw.staging.`+table+` WHERE tgl_entri <= $1 AND status = $2 AND pesan ILIKE $3 ORDER BY kode DESC LIMIT $4`, end, status, "%"+pesan+"%", limit)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var ks []int64
		for rows.Next() {
			var k int64
			if err := rows.Scan(&k); err != nil {
				return nil, err
			}
			ks = append(ks, k)
		}
		return ks, rows.Err()
	}

	check := func(table string, get func(cursor int64) ([]int64, int64, error)) {
		want, err := legacy(table, 6)
		if err != nil {
			t.Fatalf("%s: referensi gagal: %v", table, err)
		}

		var got []int64
		seen := map[int64]bool{}
		cursor := int64(0)
		for p := 0; p < 2; p++ {
			ks, next, err := get(cursor)
			if err != nil {
				t.Fatalf("%s: halaman %d gagal: %v", table, p+1, err)
			}
			for _, k := range ks {
				if seen[k] {
					t.Fatalf("%s: kode duplikat antar halaman: %d", table, k)
				}
				seen[k] = true
			}
			got = append(got, ks...)
			cursor = next
		}

		if len(got) != len(want) {
			t.Fatalf("%s: jumlah baris beda: paginate=%d referensi=%d", table, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: kode ke-%d beda: paginate=%d referensi=%d", table, i, got[i], want[i])
			}
		}
	}

	check("inbox", func(cursor int64) ([]int64, int64, error) {
		data, _, err := pgIn.Get(ctx, "maxtop", domain.InboxFilter{PageSize: 3, EndDate: &end, Status: &status, Pesan: pesan, Cursor: cursor})
		if err != nil {
			return nil, 0, err
		}
		ks := make([]int64, len(data))
		for i, d := range data {
			ks[i] = d.Kode
		}
		var next int64
		if len(data) > 0 {
			next = data[len(data)-1].Kode
		}
		return ks, next, nil
	})

	check("outbox", func(cursor int64) ([]int64, int64, error) {
		data, _, err := pgOut.Get(ctx, "maxtop", domain.OutboxFilter{PageSize: 3, EndDate: &end, Status: &status, Pesan: pesan, Cursor: cursor})
		if err != nil {
			return nil, 0, err
		}
		ks := make([]int64, len(data))
		for i, d := range data {
			ks[i] = d.Kode
		}
		var next int64
		if len(data) > 0 {
			next = data[len(data)-1].Kode
		}
		return ks, next, nil
	})
}

func TestPaginationLimitTotal(t *testing.T) {
	ready(t)
	ctx := context.Background()
	uIn := usecase.NewInboxUsecase(pgIn, msIn)
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	limit := 4

	seen := map[int64]bool{}
	p1, hasNext1, err := uIn.GetInbox(ctx, "maxtop", "postgres", domain.InboxFilter{PageSize: 2, LimitTotal: &limit, EndDate: &end})
	if err != nil {
		t.Fatalf("page1: %v", err)
	}
	if len(p1) != 2 || !hasNext1 {
		t.Fatalf("page1 harus 2 item & has_next=true, dapat %d/%v", len(p1), hasNext1)
	}
	var next1 int64
	if len(p1) > 0 {
		next1 = p1[len(p1)-1].Kode
	}
	for _, d := range p1 {
		seen[d.Kode] = true
	}

	p2, hasNext2, err := uIn.GetInbox(ctx, "maxtop", "postgres", domain.InboxFilter{PageSize: 2, LimitTotal: &limit, EndDate: &end, Cursor: next1})
	if err != nil {
		t.Fatalf("page2: %v", err)
	}
	if len(p2) != 2 || hasNext2 {
		t.Fatalf("page2 harus 2 item & has_next=false (limit tercapai), dapat %d/%v", len(p2), hasNext2)
	}
	for _, d := range p2 {
		if seen[d.Kode] {
			t.Fatalf("duplikat kode antar halaman: %d", d.Kode)
		}
		seen[d.Kode] = true
	}
	if len(seen) != limit {
		t.Fatalf("total harus %d, dapat %d", limit, len(seen))
	}

	if next1 > 0 {
		p3, hasNext3, err := uIn.GetInbox(ctx, "maxtop", "postgres", domain.InboxFilter{PageSize: 2, LimitTotal: &limit, EndDate: &end, Cursor: p2[len(p2)-1].Kode})
		if err != nil {
			t.Fatalf("page3: %v", err)
		}
		if len(p3) != 0 || hasNext3 {
			t.Fatalf("setelah limit tercapai harus kosong & has_next=false, dapat %d/%v", len(p3), hasNext3)
		}
	}
}

func TestJawabanVariantMatchesLegacy(t *testing.T) {
	ready(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 23, 23, 59, 59, 0, time.UTC)
	jawaban := true

	legacy := func(tbl string) ([]int64, error) {
		rows, err := rawPG.Query(ctx, `SELECT kode FROM pandora_dw.staging.`+tbl+` WHERE tgl_entri >= $1 AND tgl_entri <= $2 AND is_jawaban = 1 ORDER BY kode DESC LIMIT 6`, start, end)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var ks []int64
		for rows.Next() {
			var k int64
			if err := rows.Scan(&k); err != nil {
				return nil, err
			}
			ks = append(ks, k)
		}
		return ks, rows.Err()
	}

	got, _, err := pgIn.Get(ctx, "maxtop", domain.InboxFilter{PageSize: 6, StartDate: &start, EndDate: &end, JawabanFromProvider: &jawaban})
	if err != nil {
		t.Fatalf("varian inbox: %v", err)
	}
	want, err := legacy("inbox")
	if err != nil {
		t.Fatalf("legacy inbox: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("inbox: jumlah beda varian=%d legacy=%d", len(got), len(want))
	}
	for i := range want {
		if got[i].Kode != want[i] {
			t.Fatalf("inbox: kode ke-%d beda varian=%d legacy=%d", i, got[i].Kode, want[i])
		}
	}
}

func TestFlagVariantsMatchLegacy(t *testing.T) {
	ready(t)
	ctx := context.Background()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 23, 23, 59, 59, 0, time.UTC)

	compare := func(tbl string, got []int64, want []int64) {
		if len(got) != len(want) {
			t.Fatalf("%s: jumlah beda varian=%d legacy=%d", tbl, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: kode ke-%d beda varian=%d legacy=%d", tbl, i, got[i], want[i])
			}
		}
	}

	request := true
	gotPG, _, err := pgIn.Get(ctx, "maxtop", domain.InboxFilter{PageSize: 6, StartDate: &start, EndDate: &end, RequestFromReseller: &request})
	if err != nil {
		t.Fatalf("PG varian request: %v", err)
	}
	wantPG, err := legacyKodes(ctx, rawPG, `SELECT kode FROM pandora_dw.staging.inbox WHERE tgl_entri >= $1 AND tgl_entri <= $2 AND kode_reseller IS NOT NULL AND is_jawaban = 0 ORDER BY kode DESC LIMIT 6`, start, end)
	if err != nil {
		t.Fatalf("PG legacy request: %v", err)
	}
	compare("inbox/request PG", kodesOf(gotPG), wantPG)

	gotMS, _, err := msIn.Get(ctx, "maxtop", domain.InboxFilter{PageSize: 6, StartDate: &start, EndDate: &end, RequestFromReseller: &request})
	if err != nil {
		t.Fatalf("MS varian request: %v", err)
	}
	wantMS, err := legacyKodesMS(ctx, rawMS, `SELECT TOP (6) kode FROM dbo.inbox WHERE tgl_entri >= @s AND tgl_entri <= @e AND kode_reseller IS NOT NULL AND is_jawaban = 0 ORDER BY kode DESC`, sql.Named("s", start), sql.Named("e", end))
	if err != nil {
		t.Fatalf("MS legacy request: %v", err)
	}
	compare("inbox/request MS", kodesOf(gotMS), wantMS)
}

func kodesOf(xs []dto.InboxItem) []int64 {
	out := make([]int64, len(xs))
	for i, x := range xs {
		out[i] = x.Kode
	}
	return out
}

func legacyKodes(ctx context.Context, db *pgxpool.Pool, q string, args ...any) ([]int64, error) {
	rows, err := db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ks []int64
	for rows.Next() {
		var k int64
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		ks = append(ks, k)
	}
	return ks, rows.Err()
}

func legacyKodesMS(ctx context.Context, db *sql.DB, q string, args ...any) ([]int64, error) {
	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ks []int64
	for rows.Next() {
		var k int64
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		ks = append(ks, k)
	}
	return ks, rows.Err()
}

func TestReadOnlyUserSelect(t *testing.T) {
	ready(t)
	ctx := context.Background()

	for name, repo := range map[string]domain.UserRepository{"pg": pgUser, "ms": msUser} {
		_, err := repo.GetByUsername(ctx, "maxtop", "zz_tidak_ada_user_999")
		if err != nil {
			if !strings.Contains(strings.ToLower(err.Error()), "no rows") {
				t.Fatalf("%s: SELECT users gagal diluar no-row: %v", name, err)
			}
		}
	}
}

func TestReadOnlyLowerBoundSemuaSumber(t *testing.T) {
	ready(t)
	ctx := context.Background()
	limit := 3

	inFilter := domain.InboxFilter{PageSize: 5, LimitTotal: &limit}
	outFilter := domain.OutboxFilter{PageSize: 5, LimitTotal: &limit}

	for name, repo := range map[string]domain.InboxRepository{"pg": pgIn, "ms": msIn} {
		got, err := repo.LowerBound(ctx, "maxtop", inFilter)
		if err != nil {
			t.Fatalf("%s inbox LowerBound tanpa tanggal: %v", name, err)
		}
		if got < 0 {
			t.Fatalf("%s inbox LowerBound negatif: %d", name, got)
		}

		akhir := time.Now().AddDate(-1, 0, 0)
		denganBatas := inFilter
		denganBatas.EndDate = &akhir
		if _, err := repo.LowerBound(ctx, "maxtop", denganBatas); err != nil {
			t.Fatalf("%s inbox LowerBound dengan batas tanggal: %v", name, err)
		}

		// Rentang waktu yang tak berisi data sama sekali harus aman: cutEnd
		// mengembalikan 0 tanpa error, bukan ErrNoRows.
		nol := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
		rentangKosong := inFilter
		rentangKosong.EndDate = &nol
		if _, err := repo.LowerBound(ctx, "maxtop", rentangKosong); err != nil {
			t.Fatalf("%s inbox LowerBound rentang kosong: %v", name, err)
		}
		if _, _, err := repo.Get(ctx, "maxtop", domain.InboxFilter{PageSize: 5, EndDate: &nol, StartDate: &nol}); err != nil {
			t.Fatalf("%s inbox Get rentang kosong: %v", name, err)
		}

		if _, err := repo.LowerBound(ctx, "tenant-tak-terdaftar", inFilter); err == nil {
			t.Fatalf("%s: tenant tak terdaftar harus ditolak", name)
		}
	}

	for name, repo := range map[string]domain.OutboxRepository{"pg": pgOut, "ms": msOut} {
		got, err := repo.LowerBound(ctx, "maxtop", outFilter)
		if err != nil {
			t.Fatalf("%s outbox LowerBound tanpa tanggal: %v", name, err)
		}
		if got < 0 {
			t.Fatalf("%s outbox LowerBound negatif: %d", name, got)
		}

		akhir := time.Now().AddDate(-1, 0, 0)
		denganBatas := outFilter
		denganBatas.EndDate = &akhir
		if _, err := repo.LowerBound(ctx, "maxtop", denganBatas); err != nil {
			t.Fatalf("%s outbox LowerBound dengan batas tanggal: %v", name, err)
		}

		nol := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)
		rentangKosong := outFilter
		rentangKosong.EndDate = &nol
		if _, err := repo.LowerBound(ctx, "maxtop", rentangKosong); err != nil {
			t.Fatalf("%s outbox LowerBound rentang kosong: %v", name, err)
		}
		if _, _, err := repo.Get(ctx, "maxtop", domain.OutboxFilter{PageSize: 5, EndDate: &nol, StartDate: &nol}); err != nil {
			t.Fatalf("%s outbox Get rentang kosong: %v", name, err)
		}

		if _, err := repo.LowerBound(ctx, "tenant-tak-terdaftar", outFilter); err == nil {
			t.Fatalf("%s: tenant tak terdaftar harus ditolak", name)
		}
	}
}

func TestReadOnlyResellerDropdown(t *testing.T) {
	ready(t)
	ctx := context.Background()

	rr := repository.NewResellerRepositories(newRegistry(t))
	for name, repo := range map[string]domain.ResellerRepository{"pg": rr.PG, "ms": rr.MS} {
		list, err := repo.ListForDropdown(ctx, "maxtop")
		if err != nil {
			t.Fatalf("%s reseller dropdown: %v", name, err)
		}
		for i, r := range list {
			if r.Kode == "" {
				t.Fatalf("%s: kode reseller kosong di indeks %d", name, i)
			}
		}
		if _, err := repo.ListForDropdown(ctx, "tenant-tak-terdaftar"); err == nil {
			t.Fatalf("%s: tenant tak terdaftar harus ditolak", name)
		}
	}
}

// Penulisan tidak pernah sampai ke database: seluruh panggilan memakai tenant
// yang tak terdaftar, jadi registry menolak lebih dulu sebelum ada query.
func TestTenantTakTerdaftarDitolakSebelumMenyentuhDatabase(t *testing.T) {
	ready(t)
	ctx := context.Background()
	const tenant = "tenant-tak-terdaftar-999"
	limit := 3

	for name, repo := range map[string]domain.InboxRepository{"pg": pgIn, "ms": msIn} {
		if _, _, err := repo.Get(ctx, tenant, domain.InboxFilter{PageSize: 5}); err == nil {
			t.Fatalf("%s inbox Get: tenant tak dikenal harus ditolak", name)
		}
		if _, err := repo.LowerBound(ctx, tenant, domain.InboxFilter{PageSize: 5, LimitTotal: &limit}); err == nil {
			t.Fatalf("%s inbox LowerBound: tenant tak dikenal harus ditolak", name)
		}
		if err := repo.Insert(ctx, tenant, domain.Inbox{Pengirim: "08", Pesan: "x"}); err == nil {
			t.Fatalf("%s inbox Insert: tenant tak dikenal harus ditolak", name)
		}
		if err := repo.Update(ctx, tenant, 1, dto.UpdateInboxRequest{}); err == nil {
			t.Fatalf("%s inbox Update: tenant tak dikenal harus ditolak", name)
		}
	}

	for name, repo := range map[string]domain.OutboxRepository{"pg": pgOut, "ms": msOut} {
		if _, _, err := repo.Get(ctx, tenant, domain.OutboxFilter{PageSize: 5}); err == nil {
			t.Fatalf("%s outbox Get: tenant tak dikenal harus ditolak", name)
		}
		if _, err := repo.LowerBound(ctx, tenant, domain.OutboxFilter{PageSize: 5, LimitTotal: &limit}); err == nil {
			t.Fatalf("%s outbox LowerBound: tenant tak dikenal harus ditolak", name)
		}
		if err := repo.Insert(ctx, tenant, domain.Outbox{Penerima: "08", Pesan: "x"}); err == nil {
			t.Fatalf("%s outbox Insert: tenant tak dikenal harus ditolak", name)
		}
		if err := repo.Update(ctx, tenant, 1, dto.UpdateOutboxRequest{}); err == nil {
			t.Fatalf("%s outbox Update: tenant tak dikenal harus ditolak", name)
		}
	}

	for name, repo := range map[string]domain.UserRepository{"pg": pgUser, "ms": msUser} {
		if _, err := repo.GetByUsername(ctx, tenant, "apa-saja"); err == nil {
			t.Fatalf("%s user GetByUsername: tenant tak dikenal harus ditolak", name)
		}
		if err := repo.Insert(ctx, tenant, domain.User{Username: "u", Password: "p", Rules: "r"}); err == nil {
			t.Fatalf("%s user Insert: tenant tak dikenal harus ditolak", name)
		}
		if kosong := ""; repo.Update(ctx, tenant, "u", &kosong, &kosong) == nil {
			t.Fatalf("%s user Update: tenant tak dikenal harus ditolak", name)
		}
	}
}

func TestReadOnlyInbox(t *testing.T) {
	ready(t)
	ctx := context.Background()
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)

	for name, repo := range map[string]domain.InboxRepository{"pg": pgIn, "ms": msIn} {
		pengirim := "0812"
		reseller := "PDR0001"
		cases := []domain.InboxFilter{
			{PageSize: 5},
			{PageSize: 5, StartDate: &start, EndDate: &end},
			{PageSize: 5, Status: pInt16(0), Pesan: "a", Pengirim: &pengirim},
			{PageSize: 5, Reseller: &reseller},
		}
		for i, f := range cases {
			data, _, err := repo.Get(ctx, "maxtop", f)
			if err != nil {
				t.Fatalf("%s inbox case %d: %v", name, i, err)
			}
			if len(data) > f.PageSize {
				t.Fatalf("%s inbox case %d: %d baris > limit %d", name, i, len(data), f.PageSize)
			}
		}

		first, _, err := repo.Get(ctx, "maxtop", domain.InboxFilter{PageSize: 1})
		if err != nil || len(first) == 0 {
			t.Logf("%s: tabel inbox kosong, lewati case cursor", name)
			continue
		}
		cursor := first[0].Kode
		if _, _, err := repo.Get(ctx, "maxtop", domain.InboxFilter{PageSize: 5, Cursor: cursor}); err != nil {
			t.Fatalf("%s inbox cursor: %v", name, err)
		}
	}
}

func TestReadOnlyOutbox(t *testing.T) {
	ready(t)
	ctx := context.Background()
	start := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2030, 12, 31, 23, 59, 59, 0, time.UTC)

	for name, repo := range map[string]domain.OutboxRepository{"pg": pgOut, "ms": msOut} {
		penerima := "0812"
		reseller := "PDR0001"
		cases := []domain.OutboxFilter{
			{PageSize: 5},
			{PageSize: 5, StartDate: &start, EndDate: &end},
			{PageSize: 5, Status: pInt16(0), Pesan: "a", Penerima: &penerima},
			{PageSize: 5, Reseller: &reseller},
		}
		for i, f := range cases {
			data, _, err := repo.Get(ctx, "maxtop", f)
			if err != nil {
				t.Fatalf("%s outbox case %d: %v", name, i, err)
			}
			if len(data) > f.PageSize {
				t.Fatalf("%s outbox case %d: %d baris > limit %d", name, i, len(data), f.PageSize)
			}
		}
	}
}
