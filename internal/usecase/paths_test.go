package usecase

import (
	"context"
	"errors"
	"testing"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type stubWriteInboxRepo struct {
	errGet   error
	errWrite error
}

func (s *stubWriteInboxRepo) Get(context.Context, string, domain.InboxFilter) ([]dto.InboxItem, error) {
	return nil, s.errGet
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
}

func (s *stubWriteOutboxRepo) Get(context.Context, string, domain.OutboxFilter) ([]dto.OutboxItem, error) {
	return nil, s.errGet
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

func TestGetInboxMenormalkanLimit(t *testing.T) {
	repo := &recordingInboxRepo{}
	u := NewInboxUsecase(repo, repo)
	ctx := context.Background()

	for _, tc := range []struct {
		masuk int
		want  int
	}{
		{0, domain.DefaultLimit},
		{-5, domain.DefaultLimit},
		{domain.MaxLimit + 500, domain.MaxLimit},
		{500000, domain.MaxLimit},
		{5, 5},
	} {
		if _, err := u.GetInbox(ctx, "tenant-ps", domain.SourcePostgres, domain.InboxFilter{Limit: tc.masuk}); err != nil {
			t.Fatalf("limit %d tidak boleh error: %v", tc.masuk, err)
		}
		if repo.filter.Limit != tc.want {
			t.Fatalf("limit %d harus jadi %d, dapat %d", tc.masuk, tc.want, repo.filter.Limit)
		}
	}
}
func TestGetInboxMeneruskanErrorRepoGet(t *testing.T) {
	boom := errors.New("query gagal")
	repo := &stubWriteInboxRepo{errGet: boom}
	u := NewInboxUsecase(repo, repo)

	if _, err := u.GetInbox(context.Background(), "t", domain.SourcePostgres, domain.InboxFilter{Limit: 5}); !errors.Is(err, boom) {
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

func TestGetOutboxMenormalkanLimit(t *testing.T) {
	repo := &recordingOutboxRepo{}
	u := NewOutboxUsecase(repo, repo)
	ctx := context.Background()

	for _, tc := range []struct {
		masuk int
		want  int
	}{
		{0, domain.DefaultLimit},
		{-9, domain.DefaultLimit},
		{domain.MaxLimit + 99, domain.MaxLimit},
		{500000, domain.MaxLimit},
		{5, 5},
	} {
		if _, err := u.GetOutbox(ctx, "tenant-ps", domain.SourcePostgres, domain.OutboxFilter{Limit: tc.masuk}); err != nil {
			t.Fatalf("limit %d tidak boleh error: %v", tc.masuk, err)
		}
		if repo.filter.Limit != tc.want {
			t.Fatalf("limit %d harus jadi %d, dapat %d", tc.masuk, tc.want, repo.filter.Limit)
		}
	}
}

func TestGetOutboxMeneruskanErrorRepoGet(t *testing.T) {
	boom := errors.New("query gagal")
	repo := &stubWriteOutboxRepo{errGet: boom}
	u := NewOutboxUsecase(repo, repo)

	if _, err := u.GetOutbox(context.Background(), "t", domain.SourcePostgres, domain.OutboxFilter{Limit: 5}); !errors.Is(err, boom) {
		t.Fatalf("error Get harus diteruskan, dapat %v", err)
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
