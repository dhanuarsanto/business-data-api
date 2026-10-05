package http

import (
	"net/url"
	"testing"
	"time"

	"go.internal/business-data-api/internal/domain"
)

func TestParseListParamsTanpaParameter(t *testing.T) {
	p, err := parseListParams(url.Values{})
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}

	if p.Limit != domain.DefaultLimit {
		t.Fatalf("limit kosong harus default, dapat %d", p.Limit)
	}
	if p.StartDate != nil || p.EndDate != nil {
		t.Fatal("tanggal kosong harus nil")
	}
}

func TestParseListParamsLimitValid(t *testing.T) {
	cases := []struct {
		masuk string
		want  int
	}{
		{"50", 50},
		{"1", 1},
		{"100", 100},
	}

	for _, c := range cases {
		p, err := parseListParams(url.Values{"limit": {c.masuk}})
		if err != nil {
			t.Fatalf("limit %q tidak boleh error: %v", c.masuk, err)
		}
		if p.Limit != c.want {
			t.Fatalf("limit %q harus jadi %d, dapat %d", c.masuk, c.want, p.Limit)
		}
	}
}

func TestParseListParamsLimitClamped(t *testing.T) {
	p, err := parseListParams(url.Values{"limit": {"99999"}})
	if err != nil {
		t.Fatalf("limit > MaxLimit tidak boleh error: %v", err)
	}
	if p.Limit != domain.MaxLimit {
		t.Fatalf("limit > MaxLimit harus clamp ke MaxLimit, dapat %d", p.Limit)
	}
}

func TestParseListParamsLimitInvalid400(t *testing.T) {
	cases := []string{"-10", "abc", "1.5"}
	for _, c := range cases {
		_, err := parseListParams(url.Values{"limit": {c}})
		if err == nil {
			t.Fatalf("limit %q harus error 400, tapi nil", c)
		}
	}
}

func TestParseListParamsLimitZeroIgnored(t *testing.T) {
	p, err := parseListParams(url.Values{"limit": {"0"}})
	if err != nil {
		t.Fatalf("limit 0 tidak boleh error: %v", err)
	}
	if p.Limit != domain.DefaultLimit {
		t.Fatalf("limit 0 harus pakai DefaultLimit, dapat %d", p.Limit)
	}
}

func TestParseListParamsParameterPagingDiabaikan(t *testing.T) {
	p, err := parseListParams(url.Values{"pageSize": {"999"}, "cursor": {"77"}})
	if err != nil {
		t.Fatalf("tidak boleh error: %v", err)
	}

	if p.Limit != domain.DefaultLimit {
		t.Fatalf("pageSize/cursor tak boleh memengaruhi limit, dapat %d", p.Limit)
	}
}

func TestParseListParamsTanggalValid(t *testing.T) {
	p, err := parseListParams(url.Values{
		"startDate": {"2026-01-15"},
		"endDate":   {"2026-01-15"},
	})
	if err != nil {
		t.Fatalf("tanggal valid tidak boleh error: %v", err)
	}

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

func TestParseListParamsTanggalInvalid400(t *testing.T) {
	cases := []struct {
		startDate string
		endDate   string
	}{
		{"15-01-2026", "2026-01-15"},
		{"2026-13-01", "2026-01-15"},
		{"abc", "2026-01-15"},
		{"2026-01-15", "01-01-2026"},
		{"2026-01-15", "abc"},
	}

	for _, c := range cases {
		_, err := parseListParams(url.Values{
			"startDate": {c.startDate},
			"endDate":   {c.endDate},
		})
		if err == nil {
			t.Fatalf("tanggal invalid startDate=%q endDate=%q harus error 400, tapi nil", c.startDate, c.endDate)
		}
	}
}

func TestParseListParamsTanggalEmptyIgnored(t *testing.T) {
	p, err := parseListParams(url.Values{
		"startDate": {""},
		"endDate":   {""},
	})
	if err != nil {
		t.Fatalf("tanggal kosong tidak boleh error: %v", err)
	}
	if p.StartDate != nil || p.EndDate != nil {
		t.Fatal("tanggal kosong harus nil")
	}
}
