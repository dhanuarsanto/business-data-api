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

func TestBuildInboxFilterPGJawabanDariProviderMenang(t *testing.T) {
	for _, pasangan := range [][2]bool{{false, true}, {true, false}, {false, false}, {true, true}} {
		request, jawaban := pasangan[0], pasangan[1]
		t.Run(fmt.Sprintf("request=%v jawaban=%v", request, jawaban), func(t *testing.T) {
			where, _, _ := buildInboxFilterPG(domain.InboxFilter{
				RequestFromReseller: &request, JawabanFromProvider: &jawaban,
			})

			switch {
			case jawaban && !strings.Contains(where, "is_jawaban = 1"):
				t.Fatalf("jawaban dari provider harus menang: %q", where)
			case !jawaban && request && !strings.Contains(where, "kode_reseller IS NOT NULL AND is_jawaban = 0"):
				t.Fatalf("request dari reseller harus dipakai: %q", where)
			case !jawaban && !request && strings.Count(where, whereBase) != 1:
				t.Fatalf("tanpa flag aktif WHERE tak boleh berubah: %q", where)
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

func TestBuildInboxFilterMSJawabanDariProviderMenang(t *testing.T) {
	request, jawaban := true, true
	where, args := buildInboxFilterMS(domain.InboxFilter{RequestFromReseller: &request, JawabanFromProvider: &jawaban})
	if !strings.Contains(where, "is_jawaban = 1") || strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("jawaban dari provider harus menang: %q", where)
	}
	if len(args) != 0 {
		t.Fatalf("flag boolean tak menambah argumen, dapat %d", len(args))
	}

	dimatikan := false
	where, _ = buildInboxFilterMS(domain.InboxFilter{JawabanFromProvider: &dimatikan, RequestFromReseller: &request})
	if !strings.Contains(where, "kode_reseller IS NOT NULL AND is_jawaban = 0") {
		t.Fatalf("jawaban dimatikan harus jatuh ke request: %q", where)
	}

	kedua := false
	where, _ = buildInboxFilterMS(domain.InboxFilter{JawabanFromProvider: &kedua, RequestFromReseller: &kedua})
	if where != whereBase {
		t.Fatalf("semua flag mati harus polos, dapat %q", where)
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

func TestBuildOutboxFilterPGPerintahProviderMenang(t *testing.T) {
	ya, tidak := true, false
	where, args, next := buildOutboxFilterPG(domain.OutboxFilter{PerintahProvider: &ya, ReplyToReseller: &tidak})
	if !strings.Contains(where, "is_perintah = 1") || strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("perintah provider harus menang: %q", where)
	}
	if len(args) != 0 || next != 1 {
		t.Fatalf("flag tak boleh menambah argumen: %v %d", args, next)
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

func TestBuildOutboxFilterMSPerintahProviderMenangDanKosong(t *testing.T) {
	ya, tidak := true, false
	where, args := buildOutboxFilterMS(domain.OutboxFilter{PerintahProvider: &ya, ReplyToReseller: &tidak})
	if !strings.Contains(where, "is_perintah = 1") || strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("perintah provider harus menang: %q", where)
	}
	if len(args) != 0 {
		t.Fatalf("flag tak menambah argumen, dapat %d", len(args))
	}

	kosong, args := buildOutboxFilterMS(domain.OutboxFilter{})
	if kosong != whereBase || len(args) != 0 {
		t.Fatalf("filter kosong harus tetap WHERE 1=1: %q %v", kosong, args)
	}
}

func TestBuildPageQueryMSJalurDefaultMemakaiTopDanTanpaAlias(t *testing.T) {
	cols := []string{"kode", "tgl_entri"}
	q, args := buildPageQueryMS(cols, "outbox", whereBase, nil, false, 51)

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

func TestBuildPageQueryMSJalurTglMempertahankanArgumenSyarat(t *testing.T) {
	cols := []string{"kode", "tgl_entri"}
	where := whereBase + " AND kode_reseller = @reseller"
	sebelum := []any{sql.Named("reseller", "R1")}

	q, args := buildPageQueryMS(cols, "outbox", where, sebelum, true, 26)

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

func TestBuildPageQueryPGMempertahankanArgumenSyaratPadaKeduaJalur(t *testing.T) {
	cols := []string{"kode"}
	where := whereBase + " AND status = $1"
	sebelum := []any{int16(2)}

	q, args := buildPageQueryPG(cols, "inbox", where, sebelum, 2, false, 11)
	if !strings.Contains(q, "SELECT kode FROM inbox"+where+" ORDER BY kode DESC LIMIT $2") {
		t.Fatalf("jalur default tak sesuai: %q", q)
	}
	if len(args) != 2 || args[0] != int16(2) || args[1] != 11 {
		t.Fatalf("jalur default: argumen tak sesuai: %v", args)
	}

	q, args = buildPageQueryPG(cols, "inbox", where, sebelum, 2, true, 11)
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
