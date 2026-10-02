package config

import (
	"net/netip"
	"strings"
	"testing"
)

func TestGetAllowedOriginsMembersihkanDanMemangkas(t *testing.T) {
	cases := []struct {
		masuk string
		want  []string
	}{
		{"http://a.test,http://b.test", []string{"http://a.test", "http://b.test"}},
		{"  http://a.test ,  ", []string{"http://a.test"}},
		{"", []string{"http://localhost:3000", "http://localhost:5173"}},
		{",,", []string{"http://localhost:3000", "http://localhost:5173"}},
	}

	for _, c := range cases {
		cfg := Config{AllowedOrigins: c.masuk}
		got := cfg.GetAllowedOrigins()
		if len(got) != len(c.want) {
			t.Fatalf("%q harus jadi %v, dapat %v", c.masuk, c.want, got)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Fatalf("%q urutan ke-%d harus %q, dapat %q", c.masuk, i, c.want[i], got[i])
			}
		}
	}
}

func TestParseTrustedProxiesCIDRDanIP(t *testing.T) {
	cfg := Config{TrustedProxies: "10.0.0.0/8, 192.168.1.5, , 10.0.0.0/8, 172.16.0.0/12"}

	got, err := cfg.ParseTrustedProxies()
	if err != nil {
		t.Fatalf("parse tidak boleh gagal: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("entri tidak dideduplikasi, harus dapat 4, dapat %d", len(got))
	}
	if got[0].String() != "10.0.0.0/8" {
		t.Fatalf("prefix pertama salah: %s", got[0])
	}
	if got[1].String() != "192.168.1.5/32" {
		t.Fatalf("IP tunggal harus jadi /32, dapat %s", got[1])
	}
	if got[3].String() != "172.16.0.0/12" {
		t.Fatalf("prefix terakhir salah: %s", got[3])
	}
}

func TestParseTrustedProxiesIPv6HarusDidaftarkanEksplisit(t *testing.T) {
	cfg := Config{TrustedProxies: "127.0.0.1,::1,fe80::/10"}

	got, err := cfg.ParseTrustedProxies()
	if err != nil {
		t.Fatalf("parse tidak boleh gagal: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("harus dapat 3 prefix, dapat %d", len(got))
	}
	if got[1].String() != "::1/128" {
		t.Fatalf("IP IPv6 tunggal harus jadi /128, dapat %s", got[1])
	}
	if got[2].String() != "fe80::/10" {
		t.Fatalf("CIDR IPv6 salah: %s", got[2])
	}

	ipv4Only := netip.MustParsePrefix("127.0.0.1/32")
	if ipv4Only.Contains(netip.MustParseAddr("::1")) {
		t.Fatal("prefix IPv4 tidak boleh mencocokkan alamat IPv6")
	}
	if !got[1].Contains(netip.MustParseAddr("::1")) {
		t.Fatal("::1/128 harus mencocokkan ::1")
	}
}

func TestParseTrustedProxiesKosong(t *testing.T) {
	cfg := Config{TrustedProxies: "  ,  "}

	got, err := cfg.ParseTrustedProxies()
	if err != nil {
		t.Fatalf("parse tidak boleh gagal: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("input kosong harus jadi nol prefix, dapat %d", len(got))
	}
}

func TestParseTrustedProxiesRusakDitolak(t *testing.T) {
	cases := []struct {
		masuk string
		pesan string
	}{
		{"10.0.0.0/99", "CIDR"},
		{"bukan-alamat", "IP"},
		{"10.0.0.0/8,zyx", "IP"},
	}

	for _, c := range cases {
		cfg := Config{TrustedProxies: c.masuk}
		_, err := cfg.ParseTrustedProxies()
		if err == nil {
			t.Fatalf("%q harus ditolak", c.masuk)
		}
		if !strings.Contains(err.Error(), c.pesan) {
			t.Fatalf("pesan untuk %q harus menyebut %q, dapat %q", c.masuk, c.pesan, err)
		}
	}
}
