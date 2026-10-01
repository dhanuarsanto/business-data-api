package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"go.internal/business-data-api/internal/domain"
)

func ptr[T any](v T) *T { return &v }

func TestBuildInboxFilterPGSemuaFilterBernomorBerurutan(t *testing.T) {
	awal := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	akhir := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)
	terminal := 7
	reseller := "R1"
	pengirim := "Budi"
	tipe := "S"
	status := int16(2)
	ya := true

	where, args, next := buildInboxFilterPG(domain.InboxFilter{
		StartDate: &awal, EndDate: &akhir, Terminal: &terminal, Reseller: &reseller,
		Pengirim: &pengirim, Tipe: &tipe, Status: &status, Pesan: "halo",
		RequestFromReseller: &ya,
	})

	want := whereBase +
		" AND tgl_entri >= $1" +
		" AND tgl_entri <= $2" +
		" AND kode_terminal = $3" +
		" AND kode_reseller = $4" +
		" AND pengirim ILIKE $5" +
		" AND tipe_pengirim = $6" +
		" AND status = $7" +
		" AND pesan ILIKE $8" +
		" AND kode_reseller IS NOT NULL AND is_jawaban = 0"
	if where != want {
		t.Fatalf("WHERE tak sesuai:\napatasi : %q\nharus  : %q", where, want)
	}
	if next != 9 {
		t.Fatalf("argumen berikutnya harus 9, dapat %d", next)
	}
	if len(args) != 8 {
		t.Fatalf("harus ada 8 argumen, dapat %d", len(args))
	}
	if args[0] != awal || args[1] != akhir || args[2] != terminal || args[3] != reseller {
		t.Fatalf("argumen tak diteruskan urut: %v", args)
	}
	if args[4] != "%Budi%" || args[7] != "%halo%" {
		t.Fatalf("pencarian sebagian harus dibungkus wildcard: %v", args)
	}
}

func TestBuildInboxFilterPGTriaseGabung(t *testing.T) {
	for _, kasus := range []struct {
		request, jawaban bool
		ingin            string
	}{
		{false, false, whereBase},
		{false, true, whereBase + " AND is_jawaban = 1"},
		{true, false, whereBase + " AND kode_reseller IS NOT NULL AND is_jawaban = 0"},
		{true, true, whereBase},
	} {
		t.Run(fmt.Sprintf("request=%v jawaban=%v", kasus.request, kasus.jawaban), func(t *testing.T) {
			request, jawaban := kasus.request, kasus.jawaban
			where, args, next := buildInboxFilterPG(domain.InboxFilter{
				RequestFromReseller: &request, JawabanFromProvider: &jawaban,
			})
			if where != kasus.ingin {
				t.Fatalf("filter triase tak sesuai:\ndapat : %q\nharus : %q", where, kasus.ingin)
			}
			if len(args) != 0 || next != 1 {
				t.Fatalf("flag triase tak boleh menambah argumen: %v %d", args, next)
			}
		})
	}
}

func TestBuildInboxFilterPGCumaSatuFilterDanKosong(t *testing.T) {
	where, args, next := buildInboxFilterPG(domain.InboxFilter{Pesan: "satu"})
	if where != whereBase+" AND pesan ILIKE $1" || next != 2 || len(args) != 1 {
		t.Fatalf("filter tunggal tak sesuai: %q %v %d", where, args, next)
	}

	empty, args, next := buildInboxFilterPG(domain.InboxFilter{})
	if empty != whereBase || next != 1 || len(args) != 0 {
		t.Fatalf("filter kosong harus tetap WHERE 1=1: %q %v %d", empty, args, next)
	}
}

func TestBuildInboxFilterMSSemuaFilterBernamaUnik(t *testing.T) {
	awal := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	akhir := time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC)
	terminal := 7
	reseller := "R1"
	pengirim := "Budi"
	tipe := "S"
	status := int16(2)
	ya := true

	where, args := buildInboxFilterMS(domain.InboxFilter{
		StartDate: &awal, EndDate: &akhir, Terminal: &terminal, Reseller: &reseller,
		Pengirim: &pengirim, Tipe: &tipe, Status: &status, Pesan: "halo",
		RequestFromReseller: &ya,
	})

	want := whereBase +
		" AND tgl_entri >= @startDate" +
		" AND tgl_entri <= @endDate" +
		" AND kode_terminal = @terminal" +
		" AND kode_reseller = @reseller" +
		" AND pengirim LIKE '%' + @pengirim + '%'" +
		" AND tipe_pengirim = @tipe" +
		" AND status = @status" +
		" AND pesan LIKE '%' + @pesan + '%'" +
		" AND kode_reseller IS NOT NULL AND is_jawaban = 0"
	if where != want {
		t.Fatalf("WHERE tak sesuai:\napatasi : %q\nharus  : %q", where, want)
	}

	terlihat := map[string]bool{}
	for _, arg := range args {
		named, ok := arg.(sql.NamedArg)
		if !ok {
			t.Fatalf("argumen MS harus bernama, dapat %T", arg)
		}
		if terlihat[named.Name] {
			t.Fatalf("nama argumen ganda: %s", named.Name)
		}
		terlihat[named.Name] = true
	}
	for _, nama := range []string{"startDate", "endDate", "terminal", "reseller", "pengirim", "tipe", "status", "pesan"} {
		if !terlihat[nama] {
			t.Fatalf("nama argumen %q tak ada, dapat %v", nama, terlihat)
		}
	}
}

func TestBuildInboxFilterMSTriaseGabung(t *testing.T) {
	for _, kasus := range []struct {
		request, jawaban bool
		ingin            string
	}{
		{false, false, whereBase},
		{false, true, whereBase + " AND is_jawaban = 1"},
		{true, false, whereBase + " AND kode_reseller IS NOT NULL AND is_jawaban = 0"},
		{true, true, whereBase},
	} {
		t.Run(fmt.Sprintf("request=%v jawaban=%v", kasus.request, kasus.jawaban), func(t *testing.T) {
			request, jawaban := kasus.request, kasus.jawaban
			where, args := buildInboxFilterMS(domain.InboxFilter{
				RequestFromReseller: &request, JawabanFromProvider: &jawaban,
			})
			if where != kasus.ingin {
				t.Fatalf("filter triase tak sesuai:\ndapat : %q\nharus : %q", where, kasus.ingin)
			}
			if len(args) != 0 {
				t.Fatalf("flag triase tak menambah argumen, dapat %d", len(args))
			}
		})
	}
}

func TestBuildOutboxFilterPGSemuaFilterBernomorBerurutan(t *testing.T) {
	awal := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	akhir := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)
	reseller := "R2"
	penerima := "Siti"
	tipe := "P"
	status := int16(3)
	ya := true

	where, args, next := buildOutboxFilterPG(domain.OutboxFilter{
		StartDate: &awal, EndDate: &akhir, Reseller: &reseller, Penerima: &penerima,
		Tipe: &tipe, Status: &status, Pesan: "balasan", ReplyToReseller: &ya,
	})

	want := whereBase +
		" AND tgl_entri >= $1" +
		" AND tgl_entri <= $2" +
		" AND kode_reseller = $3" +
		" AND penerima ILIKE $4" +
		" AND tipe_penerima = $5" +
		" AND status = $6" +
		" AND pesan ILIKE $7" +
		" AND kode_reseller IS NOT NULL AND is_perintah = 0"
	if where != want {
		t.Fatalf("WHERE tak sesuai:\napatasi : %q\nharus  : %q", where, want)
	}
	if next != 8 || len(args) != 7 {
		t.Fatalf("penomoran argumen tak sesuai: next=%d len=%d", next, len(args))
	}
	if args[3] != "%Siti%" || args[6] != "%balasan%" {
		t.Fatalf("pencarian sebagian harus dibungkus wildcard: %v", args)
	}
}

func TestBuildOutboxFilterPGTriaseGabungDanKosong(t *testing.T) {
	for _, kasus := range []struct {
		reply, perintah bool
		ingin           string
	}{
		{false, false, whereBase},
		{false, true, whereBase + " AND is_perintah = 1"},
		{true, false, whereBase + " AND kode_reseller IS NOT NULL AND is_perintah = 0"},
		{true, true, whereBase},
	} {
		t.Run(fmt.Sprintf("reply=%v perintah=%v", kasus.reply, kasus.perintah), func(t *testing.T) {
			reply, perintah := kasus.reply, kasus.perintah
			where, args, next := buildOutboxFilterPG(domain.OutboxFilter{
				PerintahProvider: &perintah, ReplyToReseller: &reply,
			})
			if where != kasus.ingin {
				t.Fatalf("filter triase tak sesuai:\ndapat : %q\nharus : %q", where, kasus.ingin)
			}
			if len(args) != 0 || next != 1 {
				t.Fatalf("flag triase tak boleh menambah argumen: %v %d", args, next)
			}
		})
	}

	kosong, args, next := buildOutboxFilterPG(domain.OutboxFilter{})
	if kosong != whereBase || len(args) != 0 || next != 1 {
		t.Fatalf("filter kosong harus tetap WHERE 1=1: %q %v %d", kosong, args, next)
	}
}

func TestBuildOutboxFilterMSSemuaFilterBernamaUnik(t *testing.T) {
	awal := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	akhir := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)
	reseller := "R2"
	penerima := "Siti"
	tipe := "P"
	status := int16(3)
	ya := true

	where, args := buildOutboxFilterMS(domain.OutboxFilter{
		StartDate: &awal, EndDate: &akhir, Reseller: &reseller, Penerima: &penerima,
		Tipe: &tipe, Status: &status, Pesan: "balasan", ReplyToReseller: &ya,
	})

	want := whereBase +
		" AND tgl_entri >= @startDate" +
		" AND tgl_entri <= @endDate" +
		" AND kode_reseller = @reseller" +
		" AND penerima LIKE '%' + @penerima + '%'" +
		" AND tipe_penerima = @tipe" +
		" AND status = @status" +
		" AND pesan LIKE '%' + @pesan + '%'" +
		" AND kode_reseller IS NOT NULL AND is_perintah = 0"
	if where != want {
		t.Fatalf("WHERE tak sesuai:\napatasi : %q\nharus  : %q", where, want)
	}

	terlihat := map[string]bool{}
	for _, arg := range args {
		named, ok := arg.(sql.NamedArg)
		if !ok {
			t.Fatalf("argumen MS harus bernama, dapat %T", arg)
		}
		if terlihat[named.Name] {
			t.Fatalf("nama argumen ganda: %s", named.Name)
		}
		terlihat[named.Name] = true
	}
	for _, nama := range []string{"startDate", "endDate", "reseller", "penerima", "tipe", "status", "pesan"} {
		if !terlihat[nama] {
			t.Fatalf("nama argumen %q tak ada, dapat %v", nama, terlihat)
		}
	}
}

func TestBuildOutboxFilterMSTriaseGabungDanKosong(t *testing.T) {
	for _, kasus := range []struct {
		reply, perintah bool
		ingin           string
	}{
		{false, false, whereBase},
		{false, true, whereBase + " AND is_perintah = 1"},
		{true, false, whereBase + " AND kode_reseller IS NOT NULL AND is_perintah = 0"},
		{true, true, whereBase},
	} {
		t.Run(fmt.Sprintf("reply=%v perintah=%v", kasus.reply, kasus.perintah), func(t *testing.T) {
			reply, perintah := kasus.reply, kasus.perintah
			where, args := buildOutboxFilterMS(domain.OutboxFilter{
				PerintahProvider: &perintah, ReplyToReseller: &reply,
			})
			if where != kasus.ingin {
				t.Fatalf("filter triase tak sesuai:\ndapat : %q\nharus : %q", where, kasus.ingin)
			}
			if len(args) != 0 {
				t.Fatalf("flag triase tak menambah argumen, dapat %d", len(args))
			}
		})
	}

	kosong, args := buildOutboxFilterMS(domain.OutboxFilter{})
	if kosong != whereBase || len(args) != 0 {
		t.Fatalf("filter kosong harus tetap WHERE 1=1: %q %v", kosong, args)
	}
}

func TestBuildListQueryMSJalurDefaultMemakaiTopDanTanpaAlias(t *testing.T) {
	cols := []string{"kode", "tgl_entri"}
	q, args := buildListQueryMS(cols, "outbox", whereBase, nil, false, 51)

	if !strings.Contains(q, "SELECT TOP (@p_limit) kode, tgl_entri FROM outbox WHERE 1=1 ORDER BY kode DESC") {
		t.Fatalf("query default salah: %q", q)
	}
	if strings.Contains(q, "JOIN (") {
		t.Fatalf("jalur default tak boleh memakai subquery: %q", q)
	}
	if len(args) != 1 || args[0] != sql.Named("p_limit", 51) {
		t.Fatalf("argumen limit salah: %v", args)
	}
}

func TestBuildListQueryMSJalurTglMempertahankanArgumenSyarat(t *testing.T) {
	cols := []string{"kode", "tgl_entri"}
	where := whereBase + " AND kode_reseller = @reseller"
	sebelum := []any{sql.Named("reseller", "R1")}

	q, args := buildListQueryMS(cols, "outbox", where, sebelum, true, 26)

	subquery := `SELECT TOP (@p_limit) kode FROM outbox` + where + ` ORDER BY tgl_entri DESC, kode DESC`
	want := "JOIN (" + subquery + ") s ON i.kode=s.kode ORDER BY i.kode DESC"
	if !strings.Contains(q, want) {
		t.Fatalf("subquery tak sesuai:\ndapat : %q\nharus mengandung : %q", q, want)
	}
	if !strings.Contains(q, "SELECT i.kode, i.tgl_entri FROM outbox i ") {
		t.Fatalf("outer query harus memakai alias i: %q", q)
	}
	if len(args) != 2 {
		t.Fatalf("argumen syarat harus dipertahankan, dapat %d", len(args))
	}
	if args[0] != sql.Named("reseller", "R1") || args[1] != sql.Named("p_limit", 26) {
		t.Fatalf("urutan argumen tak sesuai: %v", args)
	}
}

func TestBuildListQueryPGMempertahankanArgumenSyaratPadaKeduaJalur(t *testing.T) {
	cols := []string{"kode"}
	where := whereBase + " AND status = $1"
	sebelum := []any{int16(2)}

	q, args := buildListQueryPG(cols, "inbox", where, sebelum, 2, false, 11)
	if !strings.Contains(q, "SELECT kode FROM inbox"+where+" ORDER BY kode DESC LIMIT $2") {
		t.Fatalf("jalur default tak sesuai: %q", q)
	}
	if len(args) != 2 || args[0] != int16(2) || args[1] != 11 {
		t.Fatalf("jalur default: argumen tak sesuai: %v", args)
	}

	q, args = buildListQueryPG(cols, "inbox", where, sebelum, 2, true, 11)
	subquery := "SELECT kode FROM inbox" + where + " ORDER BY tgl_entri DESC, kode DESC LIMIT $2"
	if !strings.Contains(q, "SELECT i.kode FROM inbox i JOIN ("+subquery+") s ON i.kode=s.kode ORDER BY i.kode DESC LIMIT $3") {
		t.Fatalf("jalur tgl_leading tak sesuai: %q", q)
	}
	if len(args) != 3 || args[0] != int16(2) || args[1] != 11 || args[2] != 11 {
		t.Fatalf("jalur tgl_leading: argumen tak sesuai: %v", args)
	}
}

func TestQualifyColsDanPrependBoundPadaMasukanKosong(t *testing.T) {
	if got := qualifyCols(nil, "i"); got != "" {
		t.Fatalf("daftar kolom kosong harus kosong, dapat %q", got)
	}
	if got := qualifyCols([]string{}, "i"); got != "" {
		t.Fatalf("daftar kolom kosong harus kosong, dapat %q", got)
	}
	if got := prependBound(whereBase, ""); got != whereBase {
		t.Fatalf("syarat kosong tak boleh mengubah WHERE, dapat %q", got)
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
