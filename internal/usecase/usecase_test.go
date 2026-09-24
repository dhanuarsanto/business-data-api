package usecase

import (
	"context"
	"errors"
	"testing"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
)

type stubUserRepo struct {
	users map[string]domain.User
}

func (s stubUserRepo) GetByUsername(ctx context.Context, tenant string, username string) (domain.User, error) {
	if u, ok := s.users[username]; ok {
		return u, nil
	}
	return domain.User{}, errors.New("not found")
}

func (s stubUserRepo) Insert(ctx context.Context, tenant string, data domain.User) error {
	return nil
}

func (s stubUserRepo) Update(ctx context.Context, tenant string, username string, password *string, rules *string) error {
	return nil
}

func TestCreateUserRejectsEmptyFields(t *testing.T) {
	u := NewAuthUsecase(stubUserRepo{}, stubUserRepo{})
	base := domain.User{Username: "budi", Password: "rahasia", Rules: "op"}

	cases := []domain.User{
		{Username: "", Password: "rahasia", Rules: "op"},
		{Username: "budi", Password: "", Rules: "op"},
		{Username: "budi", Password: "rahasia", Rules: ""},
	}
	for _, c := range cases {
		if err := u.CreateUser(context.Background(), "maxtop", "postgres", c); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("CreateUser dengan field kosong harus tolak: %s %s %s", c.Username, c.Password, c.Rules)
		}
	}
	if err := u.CreateUser(context.Background(), "maxtop", "postgres", base); err != nil {
		t.Fatalf("CreateUser valid harus lolos, dapat: %v", err)
	}
}

func TestUpdateUserRejectsEmptyProvidedPassword(t *testing.T) {
	u := NewAuthUsecase(stubUserRepo{}, stubUserRepo{})
	empty := ""

	if err := u.UpdateUser(context.Background(), "maxtop", "postgres", "budi", &empty, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Update dengan password kosong harus tolak, dapat: %v", err)
	}
	if err := u.UpdateUser(context.Background(), "maxtop", "postgres", "budi", nil, nil); err != nil {
		t.Fatalf("Update tanpa field (nil) harus lolos, dapat: %v", err)
	}
}

func TestLogin(t *testing.T) {
	repo := stubUserRepo{users: map[string]domain.User{
		"budi": {UserID: 1, Username: "budi", Password: "rahasia", Rules: "sa"},
	}}
	u := NewAuthUsecase(repo, repo)
	ctx := context.Background()

	user, err := u.Login(ctx, "maxtop", "postgres", "budi", "rahasia")
	if err != nil || user.UserID != 1 || user.Rules != "sa" {
		t.Fatalf("login benar harus sukses, dapat err=%v user=%+v", err, user)
	}

	if _, err := u.Login(ctx, "maxtop", "postgres", "budi", "salah"); err == nil {
		t.Fatal("password salah harus ditolak")
	}

	if _, err := u.Login(ctx, "maxtop", "postgres", "tidak-ada", "rahasia"); err == nil {
		t.Fatal("user tak dikenal harus ditolak")
	}

	if _, err := u.Login(ctx, "maxtop", "mysql", "budi", "rahasia"); err == nil {
		t.Fatal("dbSource tak dikenal harus ditolak")
	}
}

type recordingInboxRepo struct {
	called bool
	filter domain.InboxFilter
}

func (r *recordingInboxRepo) Get(ctx context.Context, tenant string, filter domain.InboxFilter) ([]domain.Inbox, bool, error) {
	r.called = true
	r.filter = filter
	return nil, false, nil
}

func (r *recordingInboxRepo) LowerBound(ctx context.Context, tenant string, filter domain.InboxFilter) (int64, error) {
	return 0, nil
}

func (r *recordingInboxRepo) Insert(ctx context.Context, tenant string, data domain.Inbox) error {
	return nil
}

func (r *recordingInboxRepo) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateInboxRequest) error {
	return nil
}

func TestInboxUsecaseLimitClamp(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		in, want int
	}{
		{0, 10},
		{-5, 10},
		{501, 10},
		{50, 50},
	} {
		repo := &recordingInboxRepo{}
		u := NewInboxUsecase(repo, repo)
		if _, _, err := u.GetInbox(ctx, "t", "postgres", domain.InboxFilter{PageSize: tc.in}); err != nil {
			t.Fatalf("limit %d error: %v", tc.in, err)
		}
		if !repo.called || repo.filter.PageSize != tc.want {
			t.Fatalf("limit %d harus di-clamp jadi %d, dapat %d", tc.in, tc.want, repo.filter.PageSize)
		}
	}

	repo := &recordingInboxRepo{}
	u := NewInboxUsecase(repo, repo)
	if _, _, err := u.GetInbox(ctx, "t", "mongo", domain.InboxFilter{PageSize: 10}); err == nil {
		t.Fatal("dbSource tak dikenal harus tolak")
	}
	if repo.called {
		t.Fatal("repo tak boleh dipanggil untuk dbSource tak dikenal")
	}
}

func TestInboxUsecaseLimitTotalCap(t *testing.T) {
	ctx := context.Background()

	limit := 5
	repoA := &recordingInboxRepo{}
	u := NewInboxUsecase(repoA, repoA)
	if _, _, err := u.GetInbox(ctx, "t", "postgres", domain.InboxFilter{PageSize: 10, LimitTotal: &limit}); err != nil {
		t.Fatalf("limit diset error: %v", err)
	}
	if !repoA.called {
		t.Fatal("repo harus dipanggil saat limit diset")
	}

	miliar := 1000000000
	repoB := &recordingInboxRepo{}
	u2 := NewInboxUsecase(repoB, repoB)
	if _, _, err := u2.GetInbox(ctx, "t", "postgres", domain.InboxFilter{PageSize: 10, LimitTotal: &miliar}); err != nil {
		t.Fatalf("limit miliaran error: %v", err)
	}
	if repoB.filter.LimitTotal == nil || *repoB.filter.LimitTotal != 100000 {
		t.Fatalf("limit miliaran harus di-clamp ke 100000, dapat %v", repoB.filter.LimitTotal)
	}
}

type recordingOutboxRepo struct {
	called bool
	filter domain.OutboxFilter
}

func (r *recordingOutboxRepo) Get(ctx context.Context, tenant string, filter domain.OutboxFilter) ([]domain.Outbox, bool, error) {
	r.called = true
	r.filter = filter
	return nil, false, nil
}

func (r *recordingOutboxRepo) LowerBound(ctx context.Context, tenant string, filter domain.OutboxFilter) (int64, error) {
	return 0, nil
}

func (r *recordingOutboxRepo) Insert(ctx context.Context, tenant string, data domain.Outbox) error {
	return nil
}

func (r *recordingOutboxRepo) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateOutboxRequest) error {
	return nil
}

func TestOutboxUsecaseLimitClamp(t *testing.T) {
	ctx := context.Background()

	for _, tc := range []struct {
		in, want int
	}{
		{0, 10},
		{501, 10},
		{25, 25},
	} {
		repo := &recordingOutboxRepo{}
		u := NewOutboxUsecase(repo, repo)
		if _, _, err := u.GetOutbox(ctx, "t", "mssql", domain.OutboxFilter{PageSize: tc.in}); err != nil {
			t.Fatalf("limit %d error: %v", tc.in, err)
		}
		if !repo.called || repo.filter.PageSize != tc.want {
			t.Fatalf("limit %d harus di-clamp jadi %d, dapat %d", tc.in, tc.want, repo.filter.PageSize)
		}
	}

	repo := &recordingOutboxRepo{}
	u := NewOutboxUsecase(repo, repo)
	if _, _, err := u.GetOutbox(ctx, "t", "oracle", domain.OutboxFilter{PageSize: 10}); err == nil {
		t.Fatal("dbSource tak dikenal harus tolak")
	}
	if repo.called {
		t.Fatal("repo tak boleh dipanggil untuk dbSource tak dikenal")
	}
}
