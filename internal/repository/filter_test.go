package repository

import (
	"strings"
	"testing"
	"time"

	"go.internal/business-data-api/internal/domain"
)

func TestBuildInboxFilterPG(t *testing.T) {
	where, args, argCount := buildInboxFilterPG(domain.InboxFilter{})
	if where != " WHERE 1=1" || len(args) != 0 || argCount != 1 {
		t.Fatalf("filter kosong salah: %q args=%d argCount=%d", where, len(args), argCount)
	}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	status := int16(3)
	where, args, argCount = buildInboxFilterPG(domain.InboxFilter{
		StartDate: &start,
		Status:    &status,
		Pesan:     "halo",
	})
	must := []string{
		" AND tgl_entri >= $1",
		" AND status = $2",
		" AND pesan ILIKE $3",
	}
	for _, s := range must {
		if !strings.Contains(where, s) {
			t.Fatalf("clause hilang %q -> %q", s, where)
		}
	}
	if argCount != 4 || len(args) != 3 {
		t.Fatalf("argCount=%d len(args)=%d", argCount, len(args))
	}
	if args[2] != "%halo%" {
		t.Fatalf("arg pesan salah: %v", args)
	}
}

func TestBuildInboxFilterPGTextLike(t *testing.T) {
	reseller := "R1"
	pengirim := "0812"
	where, args, _ := buildInboxFilterPG(domain.InboxFilter{
		Reseller: &reseller,
		Pengirim: &pengirim,
	})
	must := []string{
		" AND kode_reseller = $1",
		" AND pengirim ILIKE $2",
	}
	for _, s := range must {
		if !strings.Contains(where, s) {
			t.Fatalf("clause hilang %q -> %q", s, where)
		}
	}
	if args[0] != "R1" || args[1] != "%0812%" {
		t.Fatalf("arg harus equality untuk reseller, ILIKE %%..%% untuk pengirim: %v", args)
	}
}

func TestBuildInboxFilterMS(t *testing.T) {
	where, args := buildInboxFilterMS(domain.InboxFilter{})
	if where != " WHERE 1=1" || len(args) != 0 {
		t.Fatalf("filter kosong salah: %q args=%d", where, len(args))
	}

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	pengirim := "0812"
	where, args = buildInboxFilterMS(domain.InboxFilter{StartDate: &start, Pengirim: &pengirim})
	if !strings.Contains(where, " AND tgl_entri >= @startDate") || !strings.Contains(where, "AND pengirim LIKE '%' + @pengirim + '%'") {
		t.Fatalf("clause MS hilang: %q", where)
	}
	if len(args) != 2 {
		t.Fatalf("named args harus 2, dapat %d", len(args))
	}
}

func TestBuildInboxFilterCheckboxPriority(t *testing.T) {
	jawaban := true
	request := true

	where, _, _ := buildInboxFilterPG(domain.InboxFilter{RequestFromReseller: &request, JawabanFromProvider: &jawaban})
	if !strings.Contains(where, "AND is_jawaban = 1") {
		t.Fatalf("jawaban true harus menambah is_jawaban: %q", where)
	}
	if strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("jawaban true harus menahan requestFromReseller: %q", where)
	}

	jawaban = false
	where, _, _ = buildInboxFilterPG(domain.InboxFilter{RequestFromReseller: &request, JawabanFromProvider: &jawaban})
	if !strings.Contains(where, "kode_reseller IS NOT NULL") || !strings.Contains(where, "is_jawaban = 0") {
		t.Fatalf("jawaban false harus menjalankan request request asli is_jawaban=0: %q", where)
	}
	if strings.Contains(where, "is_jawaban = 1") {
		t.Fatalf("jawaban false tidak boleh menambah is_jawaban: %q", where)
	}
}

func TestBuildOutboxFilterCheckboxPriority(t *testing.T) {
	perintah := true
	reply := true

	where, _, _ := buildOutboxFilterPG(domain.OutboxFilter{PerintahProvider: &perintah, ReplyToReseller: &reply})
	if !strings.Contains(where, "AND is_perintah = 1") {
		t.Fatalf("perintah true harus menambah is_perintah: %q", where)
	}
	if strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("perintah true harus menahan replyToReseller: %q", where)
	}

	perintah = false
	where, _, _ = buildOutboxFilterPG(domain.OutboxFilter{PerintahProvider: &perintah, ReplyToReseller: &reply})
	if !strings.Contains(where, "kode_reseller IS NOT NULL") || !strings.Contains(where, "is_perintah = 0") {
		t.Fatalf("perintah false harus menjalankan reply request is_perintah=0: %q", where)
	}
	if strings.Contains(where, "is_perintah = 1") {
		t.Fatalf("perintah false tidak boleh menambah is_perintah: %q", where)
	}
}

func TestBuildOutboxFilterPG(t *testing.T) {
	where, _, argCount := buildOutboxFilterPG(domain.OutboxFilter{})
	if where != " WHERE 1=1" || argCount != 1 {
		t.Fatalf("filter kosong salah: %q argCount=%d", where, argCount)
	}

	penerima := "0812"
	reseller := "R1"
	where, args, argCount := buildOutboxFilterPG(domain.OutboxFilter{Penerima: &penerima, Reseller: &reseller})
	if !strings.Contains(where, " AND kode_reseller = $1") || !strings.Contains(where, " AND penerima ILIKE $2") {
		t.Fatalf("clause hilang: %q", where)
	}
	if argCount != 3 || len(args) != 2 {
		t.Fatalf("argCount=%d len(args)=%d", argCount, len(args))
	}
	if args[0] != "R1" || args[1] != "%0812%" {
		t.Fatalf("arg harus equality untuk reseller, ILIKE %%..%% untuk penerima: %v", args)
	}
}

func TestBuildOutboxFilterMS(t *testing.T) {
	penerima := "0812"
	where, args := buildOutboxFilterMS(domain.OutboxFilter{Penerima: &penerima})
	if !strings.Contains(where, "AND penerima LIKE '%' + @penerima + '%'") {
		t.Fatalf("clause hilang: %q", where)
	}
	if len(args) != 1 {
		t.Fatalf("named args harus 1, dapat %d", len(args))
	}
}
