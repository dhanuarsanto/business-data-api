package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type countingInboxRepo struct {
	recordingInboxRepo
	inserts int
	updates int
	lbCalls int
	lbValue int64
	lbErr   error
}

func (r *countingInboxRepo) Insert(ctx context.Context, tenant string, data domain.Inbox) error {
	r.inserts++
	return nil
}

func (r *countingInboxRepo) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateInboxRequest) error {
	r.updates++
	return nil
}

func (r *countingInboxRepo) LowerBound(ctx context.Context, tenant string, filter domain.InboxFilter) (int64, error) {
	r.lbCalls++
	return r.lbValue, r.lbErr
}

type failingWriteInboxRepo struct{ countingInboxRepo }

func (r *failingWriteInboxRepo) Insert(ctx context.Context, tenant string, data domain.Inbox) error {
	return errors.New("insert gagal")
}

func (r *failingWriteInboxRepo) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateInboxRequest) error {
	return errors.New("update gagal")
}

func TestCreateInboxMemanggilRepoDanMenginvalidasiCache(t *testing.T) {
	cacheSet("inbox|postgres|tenant-a|5|uji", 42)

	repo := &countingInboxRepo{}
	u := NewInboxUsecase(repo, repo)
	if err := u.CreateInbox(context.Background(), "tenant-a", domain.SourcePostgres, domain.Inbox{Pesan: "halo"}); err != nil {
		t.Fatalf("create harus sukses: %v", err)
	}
	if repo.inserts != 1 {
		t.Fatalf("repo harus dipanggil sekali, dapat %d", repo.inserts)
	}
	if _, ok := cacheGet("inbox|postgres|tenant-a|5|uji"); ok {
		t.Fatal("cache tenant yang sama harus diinvalidasi")
	}
}

func TestCreateInboxMenolakSumberTidakValid(t *testing.T) {
	repo := &countingInboxRepo{}
	u := NewInboxUsecase(repo, repo)
	if err := u.CreateInbox(context.Background(), "t", "oracle", domain.Inbox{}); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}
	if repo.inserts != 0 {
		t.Fatal("repo tak boleh dipanggil untuk sumber tak dikenal")
	}

	bad := &failingWriteInboxRepo{}
	u2 := NewInboxUsecase(bad, bad)
	if err := u2.CreateInbox(context.Background(), "t", domain.SourcePostgres, domain.Inbox{}); err == nil {
		t.Fatal("error insert harus diteruskan")
	}
}

func TestUpdateInboxMemanggilRepoDanMenginvalidasiCache(t *testing.T) {
	cacheSet("inbox|mssql|tenant-b|7|uji", 99)

	repo := &countingInboxRepo{}
	u := NewInboxUsecase(repo, repo)
	if err := u.UpdateInbox(context.Background(), "tenant-b", domain.SourceMSSQL, 5, dto.UpdateInboxRequest{}); err != nil {
		t.Fatalf("update harus sukses: %v", err)
	}
	if repo.updates != 1 {
		t.Fatalf("repo harus dipanggil sekali, dapat %d", repo.updates)
	}
	if _, ok := cacheGet("inbox|mssql|tenant-b|7|uji"); ok {
		t.Fatal("cache tenant yang sama harus diinvalidasi")
	}

	bad := &failingWriteInboxRepo{}
	u2 := NewInboxUsecase(bad, bad)
	if err := u2.UpdateInbox(context.Background(), "t", domain.SourcePostgres, 5, dto.UpdateInboxRequest{}); err == nil {
		t.Fatal("error update harus diteruskan")
	}
}

func TestGetInboxMemakaiCacheLowerBoundSekaliSaja(t *testing.T) {
	limit := 5
	repo := &countingInboxRepo{lbValue: 777}
	u := NewInboxUsecase(repo, repo)

	filter := domain.InboxFilter{PageSize: 10, LimitTotal: &limit}
	for i := 0; i < 3; i++ {
		if _, _, err := u.GetInbox(context.Background(), "tenant-cache", domain.SourcePostgres, filter); err != nil {
			t.Fatalf("panggilan %d error: %v", i, err)
		}
	}
	if repo.lbCalls != 1 {
		t.Fatalf("LowerBound harus di-cache, repo dipanggil %d kali", repo.lbCalls)
	}
	if repo.filter.LowerBound != 777 {
		t.Fatalf("LowerBound hasil repo harus terpakai, dapat %d", repo.filter.LowerBound)
	}
}

func TestGetInboxMenyembunyikanErrorLowerBound(t *testing.T) {
	limit := 5
	repo := &countingInboxRepo{lbErr: errors.New("lower bound gagal")}
	u := NewInboxUsecase(repo, repo)

	_, _, err := u.GetInbox(context.Background(), "tenant-cache2", domain.SourcePostgres, domain.InboxFilter{PageSize: 10, LimitTotal: &limit})
	if err == nil {
		t.Fatal("error LowerBound harus diteruskan")
	}
	if repo.called {
		t.Fatal("Get tak boleh jalan bila LowerBound gagal")
	}
}

func TestGetInboxLimitTotalTidakPositifJadiNil(t *testing.T) {
	nol := 0
	repo := &countingInboxRepo{}
	u := NewInboxUsecase(repo, repo)

	_, _, err := u.GetInbox(context.Background(), "t", domain.SourcePostgres, domain.InboxFilter{PageSize: 10, LimitTotal: &nol})
	if err != nil {
		t.Fatalf("limit nol tidak boleh error: %v", err)
	}
	if repo.lbCalls != 0 {
		t.Fatal("limit non-positif harus dinormalkan jadi nil, LowerBound tak boleh dipanggil")
	}
}

type countingOutboxRepo struct {
	recordingOutboxRepo
	inserts int
	updates int
	lbCalls int
}

func (r *countingOutboxRepo) Insert(ctx context.Context, tenant string, data domain.Outbox) error {
	r.inserts++
	return nil
}

func (r *countingOutboxRepo) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateOutboxRequest) error {
	r.updates++
	return nil
}

func (r *countingOutboxRepo) LowerBound(ctx context.Context, tenant string, filter domain.OutboxFilter) (int64, error) {
	r.lbCalls++
	return 100, nil
}

func TestCreateUpdateOutboxDanInvalidasiCache(t *testing.T) {
	cacheSet("outbox|postgres|tenant-c|9|uji", 1)

	repo := &countingOutboxRepo{}
	u := NewOutboxUsecase(repo, repo)
	if err := u.CreateOutbox(context.Background(), "tenant-c", domain.SourcePostgres, domain.Outbox{Pesan: "halo"}); err != nil {
		t.Fatalf("create outbox harus sukses: %v", err)
	}
	if err := u.UpdateOutbox(context.Background(), "tenant-c", domain.SourcePostgres, 3, dto.UpdateOutboxRequest{}); err != nil {
		t.Fatalf("update outbox harus sukses: %v", err)
	}
	if repo.inserts != 1 || repo.updates != 1 {
		t.Fatalf("repo harus dipanggil, insert=%d update=%d", repo.inserts, repo.updates)
	}
	if _, ok := cacheGet("outbox|postgres|tenant-c|9|uji"); ok {
		t.Fatal("cache outbox harus diinvalidasi")
	}
}

func TestCreateUpdateOutboxMenolakSumberTidakValid(t *testing.T) {
	repo := &countingOutboxRepo{}
	u := NewOutboxUsecase(repo, repo)
	if err := u.CreateOutbox(context.Background(), "t", "oracle", domain.Outbox{}); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("create harus tolak sumber salah, dapat %v", err)
	}
	if err := u.UpdateOutbox(context.Background(), "t", "oracle", 1, dto.UpdateOutboxRequest{}); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("update harus tolak sumber salah, dapat %v", err)
	}
}

func TestGetOutboxMemakaiCacheLowerBoundSekaliSaja(t *testing.T) {
	limit := 5
	repo := &countingOutboxRepo{}
	u := NewOutboxUsecase(repo, repo)

	filter := domain.OutboxFilter{PageSize: 10, LimitTotal: &limit}
	for i := 0; i < 3; i++ {
		if _, _, err := u.GetOutbox(context.Background(), "tenant-outbox-cache", domain.SourcePostgres, filter); err != nil {
			t.Fatalf("panggilan %d error: %v", i, err)
		}
	}
	if repo.lbCalls != 1 {
		t.Fatalf("LowerBound harus di-cache, repo dipanggil %d kali", repo.lbCalls)
	}
}

func TestGetOutboxLimitTotalTidakPositifJadiNil(t *testing.T) {
	negatif := -3
	repo := &countingOutboxRepo{}
	u := NewOutboxUsecase(repo, repo)

	if _, _, err := u.GetOutbox(context.Background(), "t", domain.SourceMSSQL, domain.OutboxFilter{PageSize: 10, LimitTotal: &negatif}); err != nil {
		t.Fatalf("limit negatif tidak boleh error: %v", err)
	}
	if repo.lbCalls != 0 {
		t.Fatal("limit non-positif harus dinormalkan jadi nil, LowerBound tak boleh dipanggil")
	}
}

func TestFpMenanganiNilaiKosongDanTerisi(t *testing.T) {
	if fp[int](nil) != "0" {
		t.Fatalf("pointer nil harus jadi 0, dapat %q", fp[int](nil))
	}
	v := int16(7)
	if fp(&v) != "7" {
		t.Fatalf("nilai terisi harus dicetak, dapat %q", fp(&v))
	}
}

func TestFilterCacheKeyMembedakanSeluruhDimensi(t *testing.T) {
	dasar := domain.OutboxFilter{Pesan: "halo"}
	limitA := 3
	limitB := 4

	kunciA := filterCacheKey("outbox", domain.SourcePostgres, "t", limitA, outboxFilterID(dasar))
	if kunciA == filterCacheKey("outbox", domain.SourcePostgres, "t", limitB, outboxFilterID(dasar)) {
		t.Fatal("limit berbeda harus menghasilkan kunci berbeda")
	}
	if kunciA == filterCacheKey("outbox", domain.SourceMSSQL, "t", limitA, outboxFilterID(dasar)) {
		t.Fatal("dbSource berbeda harus menghasilkan kunci berbeda")
	}
	if kunciA == filterCacheKey("outbox", domain.SourcePostgres, "t2", limitA, outboxFilterID(dasar)) {
		t.Fatal("tenant berbeda harus menghasilkan kunci berbeda")
	}
	if kunciA == filterCacheKey("inbox", domain.SourcePostgres, "t", limitA, outboxFilterID(dasar)) {
		t.Fatal("prefix modul berbeda harus menghasilkan kunci berbeda")
	}

	ada := true
	denganFlag := dasar
	denganFlag.ReplyToReseller = &ada
	if kunciA == filterCacheKey("outbox", domain.SourcePostgres, "t", limitA, outboxFilterID(denganFlag)) {
		t.Fatal("filter flag harus ikut memengaruhi kunci cache")
	}
	if kunciA == filterCacheKey("outbox", domain.SourcePostgres, "t", limitA, outboxFilterID(domain.OutboxFilter{Pesan: "berbeda"})) {
		t.Fatal("pesan berbeda harus menghasilkan kunci berbeda")
	}
}

func TestInboxFilterIDMembedakanSeluruhDimensi(t *testing.T) {
	dasar := domain.InboxFilter{Pesan: "halo"}
	dasarID := inboxFilterID(dasar)

	if dasarID == inboxFilterID(domain.InboxFilter{Pesan: "berbeda"}) {
		t.Fatal("pesan berbeda harus menghasilkan kunci berbeda")
	}

	ada := true
	denganFlag := dasar
	denganFlag.RequestFromReseller = &ada
	if dasarID == inboxFilterID(denganFlag) {
		t.Fatal("filter requestFromReseller harus ikut memengaruhi kunci cache")
	}
}

func TestCacheGetMenggantibawahNilaiKadaluarsa(t *testing.T) {
	boundCache.Store("kunci-lama", boundEntry{val: 11, at: time.Now().Add(-2 * boundTTL)})
	if _, ok := cacheGet("kunci-lama"); ok {
		t.Fatal("entri kedaluwarsa tak boleh dikembalikan")
	}
	if _, masihAda := boundCache.Load("kunci-lama"); masihAda {
		t.Fatal("entri kedaluwarsa harus dihapus dari cache")
	}
}

func TestFpStringKosong(t *testing.T) {
	s := ""
	if fp(&s) != "" {
		t.Fatalf("string kosong harus tercetak kosong, dapat %q", fp(&s))
	}
}
