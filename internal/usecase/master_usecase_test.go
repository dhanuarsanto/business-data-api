package usecase

import (
	"context"
	"errors"
	"testing"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type stubResellerRepo struct {
	items []dto.ResellerDropdown
	err   error
	calls int
}

func (s *stubResellerRepo) ListForDropdown(ctx context.Context, tenant string) ([]dto.ResellerDropdown, error) {
	s.calls++
	return s.items, s.err
}

func TestPickResellerRepoHanyaDuaSumberYangValid(t *testing.T) {
	pg := &stubResellerRepo{}
	ms := &stubResellerRepo{}

	got, err := pickResellerRepo(domain.SourcePostgres, pg, ms)
	if err != nil || got != domain.ResellerRepository(pg) {
		t.Fatalf("postgres harus mengembalikan repo pg, dapat %v %v", got, err)
	}
	got, err = pickResellerRepo(domain.SourceMSSQL, pg, ms)
	if err != nil || got != domain.ResellerRepository(ms) {
		t.Fatalf("mssql harus mengembalikan repo ms, dapat %v %v", got, err)
	}
	if _, err = pickResellerRepo("oracle", pg, ms); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}
}

func TestListResellerForDropdownMeneruskanError(t *testing.T) {
	repo := &stubResellerRepo{items: []dto.ResellerDropdown{{Kode: "R1", Nama: "Reseller Satu"}}}
	u := NewMasterUsecase(repo, repo)

	got, err := u.ListResellerForDropdown(context.Background(), "maxtop", domain.SourcePostgres)
	if err != nil {
		t.Fatalf("dropdown harus sukses: %v", err)
	}
	if len(got) != 1 || got[0].Kode != "R1" {
		t.Fatalf("dropdown harus diteruskan utuh: %+v", got)
	}

	gagal := errors.New("koneksi putus")
	u2 := NewMasterUsecase(&stubResellerRepo{err: gagal}, nil)
	if _, err := u2.ListResellerForDropdown(context.Background(), "maxtop", domain.SourcePostgres); !errors.Is(err, gagal) {
		t.Fatalf("error repo harus diteruskan, dapat %v", err)
	}
	if _, err := u2.ListResellerForDropdown(context.Background(), "maxtop", "oracle"); !errors.Is(err, domain.ErrSourceNotValid) {
		t.Fatalf("sumber tak dikenal harus ditolak, dapat %v", err)
	}
}
