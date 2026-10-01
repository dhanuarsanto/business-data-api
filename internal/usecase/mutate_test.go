package usecase

import (
	"context"
	"errors"
	"testing"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type countingInboxRepo struct {
	recordingInboxRepo
	inserts int
	updates int
}

func (r *countingInboxRepo) Insert(ctx context.Context, tenant string, data domain.Inbox) error {
	r.inserts++
	r.tenant = tenant
	return nil
}

func (r *countingInboxRepo) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateInboxRequest) error {
	r.updates++
	r.tenant = tenant
	return nil
}

type failingWriteInboxRepo struct{ countingInboxRepo }

func (r *failingWriteInboxRepo) Insert(ctx context.Context, tenant string, data domain.Inbox) error {
	return errors.New("insert gagal")
}

func (r *failingWriteInboxRepo) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateInboxRequest) error {
	return errors.New("update gagal")
}

func TestCreateInboxMemanggilRepoDanMeneruskanTenant(t *testing.T) {
	repo := &countingInboxRepo{}
	u := NewInboxUsecase(repo, repo)
	if err := u.CreateInbox(context.Background(), "tenant-a", domain.SourcePostgres, domain.Inbox{Pesan: "halo"}); err != nil {
		t.Fatalf("create harus sukses: %v", err)
	}
	if repo.inserts != 1 {
		t.Fatalf("repo harus dipanggil sekali, dapat %d", repo.inserts)
	}
	if repo.tenant != "tenant-a" {
		t.Fatalf("tenant harus diteruskan, dapat %q", repo.tenant)
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

func TestUpdateInboxMemanggilRepoDanMeneruskanTenant(t *testing.T) {
	repo := &countingInboxRepo{}
	u := NewInboxUsecase(repo, repo)
	if err := u.UpdateInbox(context.Background(), "tenant-b", domain.SourceMSSQL, 5, dto.UpdateInboxRequest{}); err != nil {
		t.Fatalf("update harus sukses: %v", err)
	}
	if repo.updates != 1 {
		t.Fatalf("repo harus dipanggil sekali, dapat %d", repo.updates)
	}
	if repo.tenant != "tenant-b" {
		t.Fatalf("tenant harus diteruskan, dapat %q", repo.tenant)
	}

	bad := &failingWriteInboxRepo{}
	u2 := NewInboxUsecase(bad, bad)
	if err := u2.UpdateInbox(context.Background(), "t", domain.SourcePostgres, 5, dto.UpdateInboxRequest{}); err == nil {
		t.Fatal("error update harus diteruskan")
	}
}

type countingOutboxRepo struct {
	recordingOutboxRepo
	inserts int
	updates int
}

func (r *countingOutboxRepo) Insert(ctx context.Context, tenant string, data domain.Outbox) error {
	r.inserts++
	r.tenant = tenant
	return nil
}

func (r *countingOutboxRepo) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateOutboxRequest) error {
	r.updates++
	r.tenant = tenant
	return nil
}

func TestCreateUpdateOutboxMemanggilRepoDanMeneruskanTenant(t *testing.T) {
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
	if repo.tenant != "tenant-c" {
		t.Fatalf("tenant harus diteruskan, dapat %q", repo.tenant)
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
