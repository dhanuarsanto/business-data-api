package repository

import (
	"strings"
	"testing"

	"go.internal/business-data-api/internal/domain"
)

func TestFilterBuildersMulaiDenganWhereBase(t *testing.T) {
	pgInbox, _, _ := buildInboxFilterPG(domain.InboxFilter{})
	msInbox, _ := buildInboxFilterMS(domain.InboxFilter{})
	pgOutbox, _, _ := buildOutboxFilterPG(domain.OutboxFilter{})
	msOutbox, _ := buildOutboxFilterMS(domain.OutboxFilter{})

	for name, got := range map[string]string{
		"buildInboxFilterPG":  pgInbox,
		"buildInboxFilterMS":  msInbox,
		"buildOutboxFilterPG": pgOutbox,
		"buildOutboxFilterMS": msOutbox,
	} {
		if !strings.HasPrefix(got, whereBase) {
			t.Fatalf("%s harus diawali %q, dapat %q", name, whereBase, got)
		}
	}
}

func TestPrependBoundMenyisipkanSetelahWhereBase(t *testing.T) {
	base, _, _ := buildInboxFilterPG(domain.InboxFilter{})
	got := prependBound(base, " AND kode <= $2")

	if !strings.HasPrefix(got, whereBase+" AND kode <= $2") {
		t.Fatalf("batas harus langsung setelah %q, dapat %q", whereBase, got)
	}
	if strings.Count(got, whereBase) != 1 {
		t.Fatalf("whereBase harus muncul sekali, dapat %q", got)
	}
	if !strings.HasSuffix(got, base[len(whereBase):]) {
		t.Fatalf("isi WHERE asli harus utuh di belakang, dapat %q", got)
	}
}

func TestQualifyColsMemberiAliasTanpaManipulasiString(t *testing.T) {
	got := qualifyCols([]string{"kode", "pesan, '') , (SELECT 1", "status"}, "i")

	want := "i.kode, i.pesan, '') , (SELECT 1, i.status"
	if got != want {
		t.Fatalf("kolom harus di-prefix per elemen, dapat %q", got)
	}
}

func TestBuildListQueryPGKodeDescUntukJalurDefault(t *testing.T) {
	cols := []string{"kode", "tgl_entri"}
	q, args := buildListQueryPG(cols, "inbox", whereBase, nil, 1, false, 21)

	if !strings.Contains(q, "SELECT kode, tgl_entri FROM inbox WHERE 1=1 ORDER BY kode DESC LIMIT $1") {
		t.Fatalf("query default salah: %q", q)
	}
	if len(args) != 1 || args[0] != 21 {
		t.Fatalf("argumen limit salah: %v", args)
	}
}

// Subquery memilih n baris terbaru berdasarkan tgl_entri; query luar wajib
// mengurutkan kode DESC supaya klien menerima kode terbesar lebih dulu dan
// tidak pernah melihat baris yang sama di dua permintaan berbeda.
func TestBuildListQueryPGOuterTerurutKodeDesc(t *testing.T) {
	cols := []string{"kode", "tgl_entri"}
	q, _ := buildListQueryPG(cols, "inbox", whereBase, nil, 1, true, 21)

	if !strings.Contains(q, "ORDER BY i.kode DESC LIMIT") {
		t.Fatalf("outer query wajib urut kode DESC, dapat: %q", q)
	}
	if !strings.Contains(q, "i.kode, i.tgl_entri") {
		t.Fatalf("kolom harus memakai alias i: %q", q)
	}
	if !strings.Contains(q, "ORDER BY tgl_entri DESC, kode DESC LIMIT $1") {
		t.Fatalf("subquery tetap harus memilih n baris terbaru berdasarkan tgl_entri: %q", q)
	}
}

func TestBuildListQueryMSOuterTerurutKodeDesc(t *testing.T) {
	cols := []string{"kode", "tgl_entri"}
	q, _ := buildListQueryMS(cols, "inbox", whereBase, nil, true, 21)

	if !strings.Contains(q, "ORDER BY i.kode DESC") {
		t.Fatalf("outer query wajib urut kode DESC, dapat: %q", q)
	}
	if !strings.Contains(q, "i.kode, i.tgl_entri") {
		t.Fatalf("kolom harus memakai alias i: %q", q)
	}
	if !strings.Contains(q, "SELECT TOP (@p_limit) kode FROM inbox WHERE 1=1 ORDER BY tgl_entri DESC, kode DESC") {
		t.Fatalf("subquery tetap harus memilih n baris terbaru berdasarkan tgl_entri: %q", q)
	}
}
