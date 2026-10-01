package http

import (
	"net/url"
	"testing"
	"time"

	"go.internal/business-data-api/internal/domain"
)

func TestParseListParamsTanpaParameter(t *testing.T) {
	p := parseListParams(url.Values{})

	if p.Limit != domain.DefaultLimit {
		t.Fatalf("limit kosong harus default, dapat %d", p.Limit)
	}
	if p.StartDate != nil || p.EndDate != nil {
		t.Fatal("tanggal kosong harus nil")
	}
}

func TestParseListParamsLimitDibatasi(t *testing.T) {
	cases := []struct {
		masuk string
		want  int
	}{
		{"99999", domain.MaxLimit},
		{"500000", domain.MaxLimit},
		{"  ", domain.DefaultLimit},
		{"-10", domain.DefaultLimit},
		{"bukan-angka", domain.DefaultLimit},
		{"50", 50},
	}

	for _, c := range cases {
		got := parseListParams(url.Values{"limit": {c.masuk}}).Limit
		if got != c.want {
			t.Fatalf("limit %q harus jadi %d, dapat %d", c.masuk, c.want, got)
		}
	}
}

func TestParseListParamsParameterPagingDiabaikan(t *testing.T) {
	p := parseListParams(url.Values{"pageSize": {"999"}, "cursor": {"77"}})

	if p.Limit != domain.DefaultLimit {
		t.Fatalf("pageSize/cursor tak boleh memengaruhi limit, dapat %d", p.Limit)
	}
}

func TestParseListParamsTanggalDivalidasiDanDigeserAkhirHari(t *testing.T) {
	p := parseListParams(url.Values{
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

func TestParseListParamsTanggalRusakDiabaikan(t *testing.T) {
	p := parseListParams(url.Values{
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
