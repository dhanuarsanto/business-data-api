package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type stubWriteInboxRepo struct {
	errGet   error
	errWrite error
}

func (s *stubWriteInboxRepo) Get(context.Context, string, domain.InboxFilter) ([]dto.InboxItem, bool, error) {
	return nil, false, s.errGet
}
func (s *stubWriteInboxRepo) LowerBound(context.Context, string, domain.InboxFilter) (int64, error) {
	return 0, nil
}
func (s *stubWriteInboxRepo) Insert(context.Context, string, domain.Inbox) error {
	return s.errWrite
}
func (s *stubWriteInboxRepo) Update(context.Context, string, int64, dto.UpdateInboxRequest) error {
	return s.errWrite
}

type stubWriteOutboxRepo struct {
	errGet   error
	errWrite error
	errLB    error
	lbCalls  int
}

func (s *stubWriteOutboxRepo) Get(context.Context, string, domain.OutboxFilter) ([]dto.OutboxItem, bool, error) {
	return nil, false, s.errGet
}
func (s *stubWriteOutboxRepo) LowerBound(context.Context, string, domain.OutboxFilter) (int64, error) {
	s.lbCalls++
	return 4242, s.errLB
}
func (s *stubWriteOutboxRepo) Insert(context.Context, string, domain.Outbox) error {
	return s.errWrite
}
func (s *stubWriteOutboxRepo) Update(context.Context, string, int64, dto.UpdateOutboxRequest) error {
	return s.errWrite
}

func TestPickUserRepoHanyaDuaSumberYangValid(t *testing.T) {
	pg := &stubWriteUserRepo{}
	ms := &stubWriteUserRepo{}

	got, err := pickUserRepo(domain.SourcePostgres, pg, ms)
	if err != nil || got != domain.UserRepository(pg) {
		t.Fatalf("postgres harus mengembalikan repo pg, dapat %v %v", got, err)
	}
	got, err = pickUserRepo(domain.SourceMSSQL, pg, ms)
	if err != nil || got != domain.UserRepository(ms) {
		t.Fatalf("mssql harus mengembalikan repo ms, dapat %v %v", got, err)
	}
	for _, salah := range []string{"", "oracle", "POSTGRES", "pg"} {
		if _, err := pickUserRepo(salah, pg, ms); !errors.Is(err, domain.ErrSourceNotValid) {
			t.Fatalf("sumber %q harus ditolak, dapat %v", salah, err)
		}
	}
}

func TestLoginMenolakSumberTidakValidSebelumMenyentuhRepo(t *testing.T) {
	repo := &stubWriteUserRepo{}
	u := NewAuthUsecase(repo, repo)

	if _, err := u.Login(context.Background(), "t", "oracle", "budi", "rahasia"); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("harus menolak sumber tak dikenal, dapat %v", err)
	}
}

type stubWriteUserRepo struct {
	writeErr error
	inserted domain.User
	updatedP *string
	updatedR *string
}

func (s *stubWriteUserRepo) GetByUsername(context.Context, string, string) (domain.User, error) {
	return domain.User{}, errors.New("tidak ditemukan")
}
func (s *stubWriteUserRepo) Insert(_ context.Context, _ string, data domain.User) error {
	s.inserted = data
	return s.writeErr
}
func (s *stubWriteUserRepo) Update(_ context.Context, _, _ string, password, rules *string) error {
	s.updatedP, s.updatedR = password, rules
	return s.writeErr
}

func TestCreateUserDanUpdateUserMeneruskanErrorRepo(t *testing.T) {
	gagal := errors.New("basis data gagal")
	repo := &stubWriteUserRepo{writeErr: gagal}
	u := NewAuthUsecase(repo, repo)
	ctx := context.Background()

	if err := u.CreateUser(ctx, "t", domain.SourcePostgres, domain.User{Username: "b", Password: "p", Rules: "r"}); !errors.Is(err, gagal) {
		t.Fatalf("error insert harus diteruskan, dapat %v", err)
	}
	if repo.inserted.Username != "b" {
		t.Fatalf("payload tak diteruskan: %+v", repo.inserted)
	}
	if err := u.UpdateUser(ctx, "t", domain.SourcePostgres, "budi", nil, nil); !errors.Is(err, gagal) {
		t.Fatalf("error update harus diteruskan, dapat %v", err)
	}
	if err := u.CreateUser(ctx, "t", "oracle", domain.User{Username: "b", Password: "p", Rules: "r"}); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}
	if err := u.UpdateUser(ctx, "t", "oracle", "budi", nil, nil); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}

	bersih := &stubWriteUserRepo{}
	u2 := NewAuthUsecase(&stubWriteUserRepo{}, bersih)
	ruangKosong := "  "
	kosong := ""
	benar := "rahasia"

	for _, c := range []struct {
		nama string
		data domain.User
	}{
		{"username spasi", domain.User{Username: ruangKosong, Password: "p", Rules: "r"}},
		{"password spasi", domain.User{Username: "b", Password: ruangKosong, Rules: "r"}},
		{"rules spasi", domain.User{Username: "b", Password: "p", Rules: ruangKosong}},
		{"username kosong", domain.User{Username: kosong, Password: "p", Rules: "r"}},
	} {
		if err := u2.CreateUser(ctx, "t", domain.SourceMSSQL, c.data); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s harus ditolak, dapat %v", c.nama, err)
		}
	}
	if err := u2.UpdateUser(ctx, "t", domain.SourceMSSQL, "budi", &kosong, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("password kosong harus ditolak, dapat %v", err)
	}
	if err := u2.UpdateUser(ctx, "t", domain.SourceMSSQL, "budi", nil, &kosong); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("rules kosong harus ditolak, dapat %v", err)
	}
	if bersih.inserted.Username != "" || bersih.updatedP != nil || bersih.updatedR != nil {
		t.Fatalf("input tak valid tak boleh menyentuh repo: %+v %v %v", bersih.inserted, bersih.updatedP, bersih.updatedR)
	}

	if err := u2.UpdateUser(ctx, "t", domain.SourceMSSQL, "budi", &benar, nil); err != nil {
		t.Fatalf("password terisi harus lolos, dapat %v", err)
	}
	if bersih.updatedP == nil || *bersih.updatedP != benar || bersih.updatedR != nil {
		t.Fatalf("hanya password yang harus diteruskan: %v %v", bersih.updatedP, bersih.updatedR)
	}
	if err := u2.UpdateUser(ctx, "t", domain.SourceMSSQL, "budi", nil, &benar); err != nil {
		t.Fatalf("rules terisi harus lolos, dapat %v", err)
	}
	if bersih.updatedR == nil || *bersih.updatedR != benar {
		t.Fatalf("rules harus diteruskan, dapat %v", bersih.updatedR)
	}
}

func TestGetInboxMenormalkanPageSizeDanLimitTotal(t *testing.T) {
	repo := &recordingInboxRepo{}
	u := NewInboxUsecase(repo, repo)
	ctx := context.Background()

	if _, _, err := u.GetInbox(ctx, "tenant-ps", domain.SourcePostgres, domain.InboxFilter{PageSize: 0}); err != nil {
		t.Fatalf("pageSize nol harus jadi default: %v", err)
	}
	if _, _, err := u.GetInbox(ctx, "tenant-ps", domain.SourcePostgres, domain.InboxFilter{PageSize: -5}); err != nil {
		t.Fatalf("pageSize negatif harus jadi default: %v", err)
	}
	if _, _, err := u.GetInbox(ctx, "tenant-ps", domain.SourcePostgres, domain.InboxFilter{PageSize: domain.MaxPageSize + 500}); err != nil {
		t.Fatalf("pageSize berlebihan harus dipangkas: %v", err)
	}

	berlebih := domain.MaxLimitTotal + 1000
	if _, _, err := u.GetInbox(ctx, "tenant-limit", domain.SourcePostgres, domain.InboxFilter{PageSize: 5, LimitTotal: &berlebih}); err != nil {
		t.Fatalf("limit berlebihan harus dipangkas: %v", err)
	}
}

func TestGetInboxMeneruskanErrorRepoGet(t *testing.T) {
	boom := errors.New("query gagal")
	repo := &stubWriteInboxRepo{errGet: boom}
	u := NewInboxUsecase(repo, repo)

	if _, _, err := u.GetInbox(context.Background(), "t", domain.SourcePostgres, domain.InboxFilter{PageSize: 5}); !errors.Is(err, boom) {
		t.Fatalf("error Get harus diteruskan, dapat %v", err)
	}
}

func TestCreateUpdateInboxMeneruskanErrorRepoDanMenolakSumberSalah(t *testing.T) {
	boom := errors.New("tulis gagal")
	repo := &stubWriteInboxRepo{errWrite: boom}
	u := NewInboxUsecase(repo, repo)
	ctx := context.Background()

	if err := u.CreateInbox(ctx, "t", domain.SourcePostgres, domain.Inbox{}); !errors.Is(err, boom) {
		t.Fatalf("error create harus diteruskan, dapat %v", err)
	}
	if err := u.UpdateInbox(ctx, "t", domain.SourceMSSQL, 1, dto.UpdateInboxRequest{}); !errors.Is(err, boom) {
		t.Fatalf("error update harus diteruskan, dapat %v", err)
	}
	if err := u.CreateInbox(ctx, "t", "oracle", domain.Inbox{}); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}
	if err := u.UpdateInbox(ctx, "t", "oracle", 1, dto.UpdateInboxRequest{}); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}
}

func TestGetOutboxMenormalkanPageSizeDanLimitTotal(t *testing.T) {
	repo := &recordingOutboxRepo{}
	u := NewOutboxUsecase(repo, repo)
	ctx := context.Background()

	if _, _, err := u.GetOutbox(ctx, "tenant-ps", domain.SourcePostgres, domain.OutboxFilter{PageSize: 0}); err != nil {
		t.Fatalf("pageSize nol harus jadi default: %v", err)
	}
	if _, _, err := u.GetOutbox(ctx, "tenant-ps", domain.SourcePostgres, domain.OutboxFilter{PageSize: domain.MaxPageSize + 99}); err != nil {
		t.Fatalf("pageSize berlebihan harus dipangkas: %v", err)
	}

	berlebih := domain.MaxLimitTotal + 1
	if _, _, err := u.GetOutbox(ctx, "tenant-limit", domain.SourcePostgres, domain.OutboxFilter{PageSize: 5, LimitTotal: &berlebih}); err != nil {
		t.Fatalf("limit berlebihan harus dipangkas: %v", err)
	}
}

func TestGetOutboxMeneruskanErrorRepoGet(t *testing.T) {
	boom := errors.New("query gagal")
	repo := &stubWriteOutboxRepo{errGet: boom}
	u := NewOutboxUsecase(repo, repo)

	if _, _, err := u.GetOutbox(context.Background(), "t", domain.SourcePostgres, domain.OutboxFilter{PageSize: 5}); !errors.Is(err, boom) {
		t.Fatalf("error Get harus diteruskan, dapat %v", err)
	}
}

func TestGetOutboxLowerBoundGagalDanBerhasilDipakai(t *testing.T) {
	tenant := "tenant-lb-" + t.Name()
	limit := 10
	filter := domain.OutboxFilter{PageSize: 5, LimitTotal: &limit}

	gagal := errors.New("batas bawah gagal")
	u := NewOutboxUsecase(&stubWriteOutboxRepo{errLB: gagal}, &stubWriteOutboxRepo{})
	if _, _, err := u.GetOutbox(context.Background(), tenant, domain.SourcePostgres, filter); !errors.Is(err, gagal) {
		t.Fatalf("error LowerBound harus diteruskan, dapat %v", err)
	}

	repo := &stubWriteOutboxRepo{}
	u2 := NewOutboxUsecase(repo, repo)
	data, _, err := u2.GetOutbox(context.Background(), tenant, domain.SourcePostgres, filter)
	if err != nil {
		t.Fatalf("LowerBound sukses tak boleh error: %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("stub mengembalikan kosong, dapat %d baris", len(data))
	}
	// Panggilan kedua harus memakai cache, bukan memanggil repo lagi.
	if _, _, err := u2.GetOutbox(context.Background(), tenant, domain.SourcePostgres, filter); err != nil {
		t.Fatalf("panggilan kedua gagal: %v", err)
	}
	if repo.lbCalls != 1 {
		t.Fatalf("LowerBound harus dipanggil sekali lalu di-cache, dapat %d", repo.lbCalls)
	}
}

func TestCreateUpdateOutboxMeneruskanErrorRepoDanMenolakSumberSalah(t *testing.T) {
	boom := errors.New("tulis gagal")
	repo := &stubWriteOutboxRepo{errWrite: boom}
	u := NewOutboxUsecase(repo, repo)
	ctx := context.Background()

	if err := u.CreateOutbox(ctx, "t", domain.SourcePostgres, domain.Outbox{}); !errors.Is(err, boom) {
		t.Fatalf("error create harus diteruskan, dapat %v", err)
	}
	if err := u.UpdateOutbox(ctx, "t", domain.SourceMSSQL, 1, dto.UpdateOutboxRequest{}); !errors.Is(err, boom) {
		t.Fatalf("error update harus diteruskan, dapat %v", err)
	}
	if err := u.CreateOutbox(ctx, "t", "oracle", domain.Outbox{}); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}
	if err := u.UpdateOutbox(ctx, "t", "oracle", 1, dto.UpdateOutboxRequest{}); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}
}

func TestPickInboxDanOutboxRepoMenolakSemuaSumberLain(t *testing.T) {
	for _, salah := range []string{"", "Oracle", "postgres ", "mysql", "POSTGRES"} {
		if _, err := pickInboxRepo(salah, nil, nil); !errors.Is(err, domain.ErrSourceNotValid) {
			t.Fatalf("pickInboxRepo %q: %v", salah, err)
		}
		if _, err := pickOutboxRepo(salah, nil, nil); !errors.Is(err, domain.ErrSourceNotValid) {
			t.Fatalf("pickOutboxRepo %q: %v", salah, err)
		}
	}
}

func TestStartBoundCacheSweepHanyaDijalankanSekali(t *testing.T) {
	startBoundCacheSweep()
	startBoundCacheSweep()
	// Goroutine penyapu dijalankan sekali saja; tunggu agar pasti ter-schedule
	// sebelum proses uji selesai, supaya pencatat cakupan tidak berlomba.
	time.Sleep(50 * time.Millisecond)
}

func TestSweepBoundCacheMembuangEntriKadaluarsaSaja(t *testing.T) {
	kunci := func(nama string) string { return "sweep-uji|" + nama + "|" + t.Name() }

	segar := kunci("segar")
	kadaluarsa := kunci("kadaluarsa")
	bukanEntri := kunci("bukan-entri")
	salahTipe := kunci("salah-tipe")

	boundCache.Store(segar, boundEntry{val: 1, at: time.Now()})
	boundCache.Store(kadaluarsa, boundEntry{val: 2, at: time.Now().Add(-2 * boundTTL)})
	boundCache.Store(bukanEntri, "bukan boundEntry")
	boundCache.Store(salahTipe, 42)
	t.Cleanup(func() {
		for _, k := range []string{segar, kadaluarsa, bukanEntri, salahTipe} {
			boundCache.Delete(k)
		}
	})

	sweepBoundCache()

	if _, ok := boundCache.Load(kadaluarsa); ok {
		t.Fatal("entri kadaluarsa harus dihapus")
	}
	if _, ok := boundCache.Load(segar); !ok {
		t.Fatal("entri segar harus dipertahankan")
	}
	if _, ok := boundCache.Load(bukanEntri); !ok {
		t.Fatal("nilai yang bukan boundEntry tak boleh dihapus")
	}
	if _, ok := boundCache.Load(salahTipe); !ok {
		t.Fatal("nilai bertipe lain tak boleh dihapus")
	}
}

func TestCacheInvalidatePrefixHanyaMenghapusAwalanCocok(t *testing.T) {
	cacheSet("inbox|postgres|tenant-a|1|satu", 1)
	cacheSet("inbox|postgres|tenant-a|1|dua", 2)
	cacheSet("inbox|mssql|tenant-a|1|satu", 3)
	cacheSet("outbox|postgres|tenant-a|1|satu", 4)

	cacheInvalidatePrefix("inbox|postgres|tenant-a")

	if _, ada := cacheGet("inbox|postgres|tenant-a|1|satu"); ada {
		t.Fatal("kunci dengan awalan cocok harus terhapus")
	}
	if _, ada := cacheGet("inbox|postgres|tenant-a|1|dua"); ada {
		t.Fatal("seluruh kunci dengan awalan cocok harus terhapus")
	}
	if _, ada := cacheGet("inbox|mssql|tenant-a|1|satu"); !ada {
		t.Fatal("kunci sumber lain harus utuh")
	}
	if _, ada := cacheGet("outbox|postgres|tenant-a|1|satu"); !ada {
		t.Fatal("kunci modul lain harus utuh")
	}
}

func TestFilterCacheKeyMemisahkanModulSumberDanTenant(t *testing.T) {
	a := filterCacheKey("inbox", "postgres", "t", 1, "id")
	b := filterCacheKey("inbox", "postgres", "t", 1, "id")
	if a != b {
		t.Fatalf("kunci harus deterministik: %q vs %q", a, b)
	}
	for _, lain := range []string{
		filterCacheKey("outbox", "postgres", "t", 1, "id"),
		filterCacheKey("inbox", "mssql", "t", 1, "id"),
		filterCacheKey("inbox", "postgres", "u", 1, "id"),
		filterCacheKey("inbox", "postgres", "t", 2, "id"),
		filterCacheKey("inbox", "postgres", "t", 1, "lain"),
	} {
		if lain == a {
			t.Fatalf("kunci tak boleh bentrok dengan %q", a)
		}
	}
	if !strings.Contains(a, "inbox|postgres|t|1|id") {
		t.Fatalf("kunci harus memuat semua dimensi: %q", a)
	}
}

func TestFpMencetakNilaiBerbagaiTipe(t *testing.T) {
	if fp[int16](nil) != "0" {
		t.Fatalf("nil harus jadi 0, dapat %q", fp[int16](nil))
	}
	if fp(&[]string{"a"}) != "[a]" {
		t.Fatalf("slice harus tercetak, dapat %q", fp(&[]string{"a"}))
	}
	angka := 3.5
	if fp(&angka) != "3.5" {
		t.Fatalf("float harus tercetak, dapat %q", fp(&angka))
	}
	benar := true
	if fp(&benar) != "true" {
		t.Fatalf("bool harus tercetak, dapat %q", fp(&benar))
	}
}
