package repository

import (
	"strings"
	"testing"

	"go.internal/business-data-api/internal/domain"
)

func TestBuildInboxFilterPGJawabanOnly(t *testing.T) {
	filter := domain.InboxFilter{
		JawabanFromProvider: ptr(true),
		RequestFromReseller: ptr(false),
	}
	where, _, _ := buildInboxFilterPG(filter)
	if !strings.Contains(where, "is_jawaban = 1") {
		t.Fatalf("WHERE harus mengandung is_jawaban = 1: %s", where)
	}
	if strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("WHERE tidak boleh mengandung kode_reseller IS NOT NULL: %s", where)
	}
}

func TestBuildInboxFilterMSJawabanOnly(t *testing.T) {
	filter := domain.InboxFilter{
		JawabanFromProvider: ptr(true),
		RequestFromReseller: ptr(false),
	}
	where, _ := buildInboxFilterMS(filter)
	if !strings.Contains(where, "is_jawaban = 1") {
		t.Fatalf("WHERE harus mengandung is_jawaban = 1: %s", where)
	}
	if strings.Contains(where, "kode_reseller IS NOT NULL") {
		t.Fatalf("WHERE tidak boleh mengandung kode_reseller IS NOT NULL: %s", where)
	}
}

func TestBuildInboxFilterPGRequestOnly(t *testing.T) {
	filter := domain.InboxFilter{
		RequestFromReseller: ptr(true),
		JawabanFromProvider: ptr(false),
	}
	where, _, _ := buildInboxFilterPG(filter)
	if !strings.Contains(where, "kode_reseller IS NOT NULL AND is_jawaban = 0") {
		t.Fatalf("WHERE harus mengandung kode_reseller IS NOT NULL AND is_jawaban = 0: %s", where)
	}
	if strings.Contains(where, "is_jawaban = 1") && !strings.Contains(where, "is_jawaban = 0") {
		t.Fatalf("WHERE tidak boleh mengandung is_jawaban = 1 tanpa 0: %s", where)
	}
}

func TestBuildInboxFilterMSRequestOnly(t *testing.T) {
	filter := domain.InboxFilter{
		RequestFromReseller: ptr(true),
		JawabanFromProvider: ptr(false),
	}
	where, _ := buildInboxFilterMS(filter)
	if !strings.Contains(where, "kode_reseller IS NOT NULL AND is_jawaban = 0") {
		t.Fatalf("WHERE harus mengandung kode_reseller IS NOT NULL AND is_jawaban = 0: %s", where)
	}
	if strings.Contains(where, "is_jawaban = 1") && !strings.Contains(where, "is_jawaban = 0") {
		t.Fatalf("WHERE tidak boleh mengandung is_jawaban = 1 tanpa 0: %s", where)
	}
}
