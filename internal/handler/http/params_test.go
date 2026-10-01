package http

import (
	"net/url"
	"testing"
	"time"

	"go.internal/business-data-api/internal/domain"
)

func TestParsePageParamsTanpaParameter(t *testing.T) {
	p := parsePageParams(url.Values{})

	if p.PageSize != domain.DefaultPageSize {
		t.Fatalf("pageSize kosong harus default, dapat %d", p.PageSize)
	}
	if p.Cursor != 0 {
		t.Fatalf("cursor kosong harus nol, dapat %d", p.Cursor)
	}
	if p.LimitTotal != nil {
		t.Fatalf("limit kosong harus nil, dapat %v", *p.LimitTotal)
	}
	if p.StartDate != nil || p.EndDate != nil {
		t.Fatalf("tanggal kosong harus nil")
	}
}

func TestParsePageParamsPageSizeDibatasi(t *testing.T) {
	cases := []struct {
		masuk string
		want  int
	}{
		{"99999", domain.MaxPageSize},
		{"  ", domain.DefaultPageSize},
		{"-10", domain.DefaultPageSize},
		{"bukan-angka", domain.DefaultPageSize},
		{"50", 50},
	}

	for _, c := range cases {
		got := parsePageParams(url.Values{"pageSize": {c.masuk}}).PageSize
		if got != c.want {
			t.Fatalf("pageSize %q harus jadi %d, dapat %d", c.masuk, c.want, got)
		}
	}
}

func TestParsePageParamsCursorRusakDiabaikan(t *testing.T) {
	for _, masuk := range []string{"bukan-angka", "", "99999999999999999999"} {
		if got := parsePageParams(url.Values{"cursor": {masuk}}).Cursor; got != 0 {
			t.Fatalf("cursor %q harus jadi nol, dapat %d", masuk, got)
		}
	}
	if got := parsePageParams(url.Values{"cursor": {"777"}}).Cursor; got != 777 {
		t.Fatalf("cursor angka harus kept, dapat %d", got)
	}
}

func TestParsePageParamsLimitDivalidasi(t *testing.T) {
	cases := []struct {
		masuk string
		ada   bool
		want  int
	}{
		{"50", true, 50},
		{"0", false, 0},
		{"-1", false, 0},
		{"bukan-angka", false, 0},
		{"", false, 0},
		{"99999999", true, domain.MaxLimitTotal},
	}

	for _, c := range cases {
		got := parsePageParams(url.Values{"limit": {c.masuk}}).LimitTotal
		if (got != nil) != c.ada {
			t.Fatalf("limit %q harus ada=%v, dapat %v", c.masuk, c.ada, got)
		}
		if got != nil && *got != c.want {
			t.Fatalf("limit %q harus jadi %d, dapat %d", c.masuk, c.want, *got)
		}
	}
}

func TestParsePageParamsTanggalDivalidasiDanDigeserAkhirHari(t *testing.T) {
	p := parsePageParams(url.Values{
		"startDate": {"2026-01-15"},
		"endDate":   {"2026-01-15"},
	})

	if p.StartDate == nil || p.StartDate.Format(dateLayout) != "2026-01-15" {
		t.Fatalf("startDate harus terparse, dapat %v", p.StartDate)
	}
	if p.EndDate == nil {
		t.Fatal("endDate harus terparse")
	}
	want := time.Date(2026, 1, 15, 23, 59, 59, 0, time.UTC)
	if !p.EndDate.Equal(want) {
		t.Fatalf("endDate harus digeser ke akhir hari %v, dapat %v", want, p.EndDate)
	}
}

func TestParsePageParamsTanggalRusakDiabaikan(t *testing.T) {
	p := parsePageParams(url.Values{
		"startDate": {"15-01-2026"},
		"endDate":   {""},
	})
	if p.StartDate != nil {
		t.Fatalf("startDate rusak harus diabaikan, dapat %v", p.StartDate)
	}
	if p.EndDate != nil {
		t.Fatalf("endDate kosong harus diabaikan, dapat %v", p.EndDate)
	}
}
