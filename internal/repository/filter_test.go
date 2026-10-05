package repository

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"go.internal/business-data-api/internal/domain"
)

func ptr[T any](v T) *T {
	return &v
}

func TestBuildInboxFilterPGBasic(t *testing.T) {
	filter := domain.InboxFilter{
		StartDate:           ptr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:             ptr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		Terminal:            ptr(5),
		Reseller:            ptr("R001"),
		Pengirim:            ptr("BANK"),
		Tipe:                ptr("S"),
		Status:              ptr(int16(0)),
		StatusMin:           ptr(int16(40)),
		Pesan:               "test pesan",
		RequestFromReseller: ptr(true),
		JawabanFromProvider: ptr(false),
	}

	where, args, argID := buildInboxFilterPG(filter)

	if !strings.Contains(where, "tgl_entri") || !strings.Contains(where, "kode_terminal") || !strings.Contains(where, "kode_reseller") || !strings.Contains(where, "pengirim") || !strings.Contains(where, "tipe_pengirim") || !strings.Contains(where, "status") || !strings.Contains(where, "pesan") || !strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("WHERE tidak lengkap: %s", where)
	}
	if argID != 10 {
		t.Fatalf("argID harus 10, got %d", argID)
	}
	if len(args) != 9 {
		t.Fatalf("args harus 9, got %d", len(args))
	}
}

func TestBuildInboxFilterMSBasic(t *testing.T) {
	filter := domain.InboxFilter{
		StartDate:           ptr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:             ptr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		Terminal:            ptr(5),
		Reseller:            ptr("R001"),
		Pengirim:            ptr("BANK"),
		Tipe:                ptr("S"),
		Status:              ptr(int16(0)),
		StatusMin:           ptr(int16(40)),
		Pesan:               "test pesan",
		RequestFromReseller: ptr(true),
		JawabanFromProvider: ptr(false),
	}

	where, args := buildInboxFilterMS(filter)

	if !strings.Contains(where, "tgl_entri") || !strings.Contains(where, "kode_terminal") || !strings.Contains(where, "kode_reseller") || !strings.Contains(where, "pengirim") || !strings.Contains(where, "tipe_pengirim") || !strings.Contains(where, "status") || !strings.Contains(where, "pesan") || !strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("WHERE tidak lengkap: %s", where)
	}
	if len(args) != 9 {
		t.Fatalf("args harus 9, got %d", len(args))
	}
}

func TestBuildOutboxFilterPGBasic(t *testing.T) {
	filter := domain.OutboxFilter{
		StartDate:        ptr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:          ptr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		Reseller:         ptr("R001"),
		Penerima:         ptr("08123456789"),
		Tipe:             ptr("P"),
		Status:           ptr(int16(0)),
		StatusMin:        ptr(int16(40)),
		Pesan:            "test pesan",
		ReplyToReseller:  ptr(true),
		PerintahProvider: ptr(false),
	}

	where, args, argID := buildOutboxFilterPG(filter)

	if !strings.Contains(where, "tgl_entri") || !strings.Contains(where, "kode_reseller") || !strings.Contains(where, "penerima") || !strings.Contains(where, "tipe_penerima") || !strings.Contains(where, "status") || !strings.Contains(where, "pesan") {
		t.Fatalf("WHERE tidak lengkap: %s", where)
	}
	if argID != 9 {
		t.Fatalf("argID harus 9, got %d", argID)
	}
	if len(args) != 8 {
		t.Fatalf("args harus 8, got %d", len(args))
	}
}

func TestBuildOutboxFilterMSBasic(t *testing.T) {
	filter := domain.OutboxFilter{
		StartDate:        ptr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndDate:          ptr(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)),
		Reseller:         ptr("R001"),
		Penerima:         ptr("08123456789"),
		Tipe:             ptr("P"),
		Status:           ptr(int16(0)),
		StatusMin:        ptr(int16(40)),
		Pesan:            "test pesan",
		ReplyToReseller:  ptr(true),
		PerintahProvider: ptr(false),
	}

	where, args := buildOutboxFilterMS(filter)

	if !strings.Contains(where, "tgl_entri") || !strings.Contains(where, "kode_reseller") || !strings.Contains(where, "penerima") || !strings.Contains(where, "tipe_penerima") || !strings.Contains(where, "status") || !strings.Contains(where, "pesan") {
		t.Fatalf("WHERE tidak lengkap: %s", where)
	}
	if len(args) != 8 {
		t.Fatalf("args harus 8, got %d", len(args))
	}
}

func TestPtrMembantuMenyusunFilterUji(t *testing.T) {
	if *ptr(5) != 5 || *ptr("teks") != "teks" {
		t.Fatal("ptr harus mengembalikan alamat nilai yang sama")
	}
	if ptr(time.Now()) == nil {
		t.Fatal("ptr waktu harus tak nil")
	}
}

func TestBuildInboxFilterPGWildcardEscape(t *testing.T) {
	tests := []struct {
		name     string
		filter   domain.InboxFilter
		wantArgs []any
	}{
		{
			name:     "pengirim with percent",
			filter:   domain.InboxFilter{Pengirim: ptr("100%")},
			wantArgs: []any{"%100\\%%"},
		},
		{
			name:     "pengirim with underscore",
			filter:   domain.InboxFilter{Pengirim: ptr("user_name")},
			wantArgs: []any{"%user\\_name%"},
		},
		{
			name:     "pengirim with both",
			filter:   domain.InboxFilter{Pengirim: ptr("test_%")},
			wantArgs: []any{"%test\\_\\%%"},
		},
		{
			name:     "pesan with percent",
			filter:   domain.InboxFilter{Pesan: "error%"},
			wantArgs: []any{"%error\\%%"},
		},
		{
			name:     "pesan with underscore",
			filter:   domain.InboxFilter{Pesan: "msg_test"},
			wantArgs: []any{"%msg\\_test%"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			where, args, _ := buildInboxFilterPG(tc.filter)
			if !strings.Contains(where, "ILIKE") {
				t.Fatalf("WHERE harus mengandung ILIKE: %s", where)
			}
			for i, want := range tc.wantArgs {
				if i >= len(args) {
					t.Fatalf("argumen ke-%d tidak ada, total %d", i, len(args))
				}
				if args[i] != want {
					t.Fatalf("argumen ke-%d: got %q, want %q", i, args[i], want)
				}
			}
		})
	}
}

func TestBuildInboxFilterMSWildcardEscape(t *testing.T) {
	tests := []struct {
		name     string
		filter   domain.InboxFilter
		wantArgs []any
	}{
		{
			name:     "pengirim with percent",
			filter:   domain.InboxFilter{Pengirim: ptr("100%")},
			wantArgs: []any{sql.Named("pengirim", "100\\%")},
		},
		{
			name:     "pengirim with underscore",
			filter:   domain.InboxFilter{Pengirim: ptr("user_name")},
			wantArgs: []any{sql.Named("pengirim", "user\\_name")},
		},
		{
			name:     "pesan with percent",
			filter:   domain.InboxFilter{Pesan: "error%"},
			wantArgs: []any{sql.Named("pesan", "error\\%")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			where, args := buildInboxFilterMS(tc.filter)
			if !strings.Contains(where, "LIKE") {
				t.Fatalf("WHERE harus mengandung LIKE: %s", where)
			}
			found := false
			for _, arg := range args {
				named, ok := arg.(sql.NamedArg)
				if !ok {
					continue
				}
				for _, want := range tc.wantArgs {
					wantNamed, _ := want.(sql.NamedArg)
					if named.Name == wantNamed.Name && named.Value == wantNamed.Value {
						found = true
						break
					}
				}
			}
			if !found {
				t.Fatalf("argumen escaped tidak ditemukan di %v", args)
			}
		})
	}
}

func TestBuildOutboxFilterPGWildcardEscape(t *testing.T) {
	tests := []struct {
		name     string
		filter   domain.OutboxFilter
		wantArgs []any
	}{
		{
			name:     "penerima with percent",
			filter:   domain.OutboxFilter{Penerima: ptr("99%")},
			wantArgs: []any{"%99\\%%"},
		},
		{
			name:     "penerima with underscore",
			filter:   domain.OutboxFilter{Penerima: ptr("user_name")},
			wantArgs: []any{"%user\\_name%"},
		},
		{
			name:     "pesan with percent",
			filter:   domain.OutboxFilter{Pesan: "done%"},
			wantArgs: []any{"%done\\%%"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			where, args, _ := buildOutboxFilterPG(tc.filter)
			if !strings.Contains(where, "ILIKE") {
				t.Fatalf("WHERE harus mengandung ILIKE: %s", where)
			}
			for i, want := range tc.wantArgs {
				if i >= len(args) {
					t.Fatalf("argumen ke-%d tidak ada, total %d", i, len(args))
				}
				if args[i] != want {
					t.Fatalf("argumen ke-%d: got %q, want %q", i, args[i], want)
				}
			}
		})
	}
}

func TestBuildOutboxFilterMSWildcardEscape(t *testing.T) {
	tests := []struct {
		name     string
		filter   domain.OutboxFilter
		wantArgs []any
	}{
		{
			name:     "penerima with percent",
			filter:   domain.OutboxFilter{Penerima: ptr("99%")},
			wantArgs: []any{sql.Named("penerima", "99\\%")},
		},
		{
			name:     "penerima with underscore",
			filter:   domain.OutboxFilter{Penerima: ptr("user_name")},
			wantArgs: []any{sql.Named("penerima", "user\\_name")},
		},
		{
			name:     "pesan with percent",
			filter:   domain.OutboxFilter{Pesan: "done%"},
			wantArgs: []any{sql.Named("pesan", "done\\%")},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			where, args := buildOutboxFilterMS(tc.filter)
			if !strings.Contains(where, "LIKE") {
				t.Fatalf("WHERE harus mengandung LIKE: %s", where)
			}
			found := false
			for _, arg := range args {
				named, ok := arg.(sql.NamedArg)
				if !ok {
					continue
				}
				for _, want := range tc.wantArgs {
					wantNamed, _ := want.(sql.NamedArg)
					if named.Name == wantNamed.Name && named.Value == wantNamed.Value {
						found = true
						break
					}
				}
			}
			if !found {
				t.Fatalf("argumen escaped tidak ditemukan di %v", args)
			}
		})
	}
}

func TestBuildInboxFilterPGStatusMin(t *testing.T) {
	filter := domain.InboxFilter{StatusMin: ptr(int16(40))}
	where, args, _ := buildInboxFilterPG(filter)
	if !strings.Contains(where, "status >=") {
		t.Fatalf("WHERE harus mengandung status >=: %s", where)
	}
	if len(args) != 1 || args[0] != int16(40) {
		t.Fatalf("argumen harus [40], got %v", args)
	}
}

func TestBuildInboxFilterMSStatusMin(t *testing.T) {
	filter := domain.InboxFilter{StatusMin: ptr(int16(40))}
	where, args := buildInboxFilterMS(filter)
	if !strings.Contains(where, "status >=") {
		t.Fatalf("WHERE harus mengandung status >=: %s", where)
	}
	found := false
	for _, arg := range args {
		named, ok := arg.(sql.NamedArg)
		if ok && named.Name == "statusMin" && named.Value == int16(40) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("argumen statusMin tidak ditemukan di %v", args)
	}
}

func TestBuildOutboxFilterPGStatusMin(t *testing.T) {
	filter := domain.OutboxFilter{StatusMin: ptr(int16(40))}
	where, args, _ := buildOutboxFilterPG(filter)
	if !strings.Contains(where, "status >=") {
		t.Fatalf("WHERE harus mengandung status >=: %s", where)
	}
	if len(args) != 1 || args[0] != int16(40) {
		t.Fatalf("argumen harus [40], got %v", args)
	}
}

func TestBuildOutboxFilterMSStatusMin(t *testing.T) {
	filter := domain.OutboxFilter{StatusMin: ptr(int16(40))}
	where, args := buildOutboxFilterMS(filter)
	if !strings.Contains(where, "status >=") {
		t.Fatalf("WHERE harus mengandung status >=: %s", where)
	}
	found := false
	for _, arg := range args {
		named, ok := arg.(sql.NamedArg)
		if ok && named.Name == "statusMin" && named.Value == int16(40) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("argumen statusMin tidak ditemukan di %v", args)
	}
}
