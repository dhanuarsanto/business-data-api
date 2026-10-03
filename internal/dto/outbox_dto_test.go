package dto

import (
	"encoding/json"
	"testing"
	"time"
)

var wantOutboxItemKeys = []string{
	"kode", "tgl_entri", "penerima", "kode_reseller", "pesan", "status", "tgl_status", "tipe_penerima", "kode_inbox",
}

func outboxItemKeys(t *testing.T, item OutboxItem) map[string]any {
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

func assertOutboxItemKeys(t *testing.T, item OutboxItem) {
	t.Helper()
	m := outboxItemKeys(t, item)
	if len(m) != len(wantOutboxItemKeys) {
		t.Fatalf("jumlah key = %d, mau %d: %v", len(m), len(wantOutboxItemKeys), m)
	}
	for _, k := range wantOutboxItemKeys {
		if _, ok := m[k]; !ok {
			t.Fatalf("key %q hilang dari response: %v", k, m)
		}
	}
}

func TestOutboxItemKeysTerlengkap(t *testing.T) {
	ts := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	rs := "R001"
	ki := int64(123)
	assertOutboxItemKeys(t, OutboxItem{
		Kode:         90210,
		TglEntri:     ts,
		Penerima:     "08123456789",
		KodeReseller: &rs,
		Pesan:        "SALDO",
		Status:       0,
		TglStatus:    &ts,
		TipePenerima: "1",
		KodeInbox:    &ki,
	})
}

func TestOutboxItemKeyNullableHilangSaatNull(t *testing.T) {
	m := outboxItemKeys(t, OutboxItem{
		Kode:         1,
		TglEntri:     time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC),
		Penerima:     "x",
		Pesan:        "y",
		TipePenerima: "",
		KodeInbox:    nil,
	})

	for _, k := range []string{"kode_reseller", "tgl_status", "kode_inbox"} {
		if v, ok := m[k]; ok {
			t.Fatalf("key %q harus hilang saat null, dapat %v", k, v)
		}
	}

	if len(m) != 6 {
		t.Fatalf("key count = %d, mau 6: %v", len(m), m)
	}
}

func TestOutboxItemKeyNotNullSelaluAda(t *testing.T) {
	for _, tc := range []struct {
		nama string
		item OutboxItem
	}{
		{"terisi", OutboxItem{Kode: 1, TglEntri: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC), Penerima: "x", Pesan: "y"}},
		{"string kosong", OutboxItem{TglEntri: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)}},
	} {
		t.Run(tc.nama, func(t *testing.T) {
			m := outboxItemKeys(t, tc.item)
			for _, k := range []string{"kode", "tgl_entri", "penerima", "pesan", "status"} {
				if _, ok := m[k]; !ok {
					t.Fatalf("key %q kolom NOT NULL harus selalu ada: %v", k, m)
				}
			}
		})
	}
}

func TestOutboxItemKolomLamaTidakBocor(t *testing.T) {
	m := outboxItemKeys(t, OutboxItem{
		Kode:     1,
		TglEntri: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC),
		Penerima: "x",
		Pesan:    "y",
	})

	for _, k := range []string{
		"kode_transaksi", "bebas_biaya",
		"is_perintah", "kode_modul", "prioritas", "modul_proses",
		"pengirim", "kode_terminal", "ctr_kirim", "nama_reseller",
	} {
		if _, ok := m[k]; ok {
			t.Fatalf("key %q tidak boleh ada di response GET outbox", k)
		}
	}
}
