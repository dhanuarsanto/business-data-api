package dto

import (
	"encoding/json"
	"testing"
	"time"
)

var wantInboxItemKeys = []string{
	"kode", "tgl_entri", "pengirim", "kode_reseller", "pesan",
	"status", "tgl_status", "kode_terminal", "service_center",
}

func inboxItemKeys(t *testing.T, item InboxItem) map[string]any {
	t.Helper()
	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func assertInboxItemKeys(t *testing.T, item InboxItem) {
	t.Helper()
	m := inboxItemKeys(t, item)
	if len(m) != len(wantInboxItemKeys) {
		t.Fatalf("jumlah key = %d, mau %d: %v", len(m), len(wantInboxItemKeys), m)
	}
	for _, k := range wantInboxItemKeys {
		if _, ok := m[k]; !ok {
			t.Fatalf("key %q hilang dari response: %v", k, m)
		}
	}
}

func TestInboxItemKeysTerlengkap(t *testing.T) {
	ts := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	rs := "R001"
	sc := "SC01"
	term := 1
	assertInboxItemKeys(t, InboxItem{
		Kode:          105423,
		TglEntri:      ts,
		Pengirim:      "08123456789",
		KodeReseller:  &rs,
		Pesan:         "SALDO",
		Status:        0,
		TglStatus:     &ts,
		KodeTerminal:  &term,
		ServiceCenter: &sc,
	})
}

func TestInboxItemKeyNullableHilangSaatNull(t *testing.T) {
	m := inboxItemKeys(t, InboxItem{Kode: 1, TglEntri: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC), Pengirim: "x", Pesan: "y"})

	for _, k := range []string{"kode_reseller", "tgl_status", "kode_terminal", "service_center"} {
		if v, ok := m[k]; ok {
			t.Fatalf("key %q harus hilang saat null, dapat %v", k, v)
		}
	}

	if len(m) != 5 {
		t.Fatalf("key count = %d, mau 5: %v", len(m), m)
	}
}

func TestInboxItemKeyNotNullSelaluAda(t *testing.T) {
	for _, tc := range []struct {
		nama string
		item InboxItem
	}{
		{"terisi", InboxItem{Kode: 1, TglEntri: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC), Pengirim: "x", Pesan: "y"}},
		{"string kosong", InboxItem{TglEntri: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)}},
	} {
		t.Run(tc.nama, func(t *testing.T) {
			m := inboxItemKeys(t, tc.item)
			for _, k := range []string{"kode", "tgl_entri", "pengirim", "pesan", "status"} {
				if _, ok := m[k]; !ok {
					t.Fatalf("key %q kolom NOT NULL harus selalu ada: %v", k, m)
				}
			}
		})
	}
}

func TestInboxItemKolomLamaTidakBocor(t *testing.T) {
	m := inboxItemKeys(t, InboxItem{Kode: 1, TglEntri: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC), Pengirim: "x", Pesan: "y"})

	for _, k := range []string{
		"tipe_pengirim", "penerima", "kode_transaksi", "is_jawaban",
		"is_cs", "kode_jawaban_cs", "hash", "nama_reseller",
	} {
		if _, ok := m[k]; ok {
			t.Fatalf("key %q tidak boleh ada di response GET inbox", k)
		}
	}
}
