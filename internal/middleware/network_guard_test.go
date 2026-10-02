package middleware

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
)

func mustResolver(t *testing.T, cidr string) *TrustedProxyResolver {
	t.Helper()
	p, err := netip.ParsePrefix(cidr)
	if err != nil {
		t.Fatalf("CIDR uji tidak valid: %v", err)
	}
	return NewTrustedProxyResolver([]netip.Prefix{p})
}

func mustResolverBanyak(t *testing.T, cidrs ...string) *TrustedProxyResolver {
	t.Helper()
	var prefixes []netip.Prefix
	for _, c := range cidrs {
		p, err := netip.ParsePrefix(c)
		if err != nil {
			t.Fatalf("CIDR uji tidak valid: %v", err)
		}
		prefixes = append(prefixes, p)
	}
	return NewTrustedProxyResolver(prefixes)
}

func TestClientAddrAbaikanForwardedForPalsuDariClient(t *testing.T) {
	resolver := mustResolver(t, "10.0.0.0/8")

	cases := []struct {
		nama     string
		remote   string
		headers  map[string]string
		expected string
	}{
		{
			nama:     "proxy menimpa XFF, client publik dipakai",
			remote:   "10.0.0.1:5000",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.7"},
			expected: "203.0.113.7",
		},
		{
			nama:     "client menyuntik XFF paling kiri,proxy nyata di kanan",
			remote:   "10.0.0.1:5000",
			headers:  map[string]string{"X-Forwarded-For": "192.168.99.1, 10.0.0.2, 198.51.100.9"},
			expected: "198.51.100.9",
		},
		{
			nama:     "seluruh rantai XFF adalah proxy-server yang dipercaya",
			remote:   "10.0.0.1:5000",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.3, 10.0.0.2"},
			expected: "10.0.0.1",
		},
		{
			nama:     "X-Real-IP dipakai hanya saat XFF tidak ada",
			remote:   "10.0.0.1:5000",
			headers:  map[string]string{"X-Real-IP": "203.0.113.7"},
			expected: "203.0.113.7",
		},
		{
			nama:     "remote tidak dipercaya, header diabaikan sepenuhnya",
			remote:   "198.51.100.9:5000",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1", "X-Real-IP": "10.0.0.1"},
			expected: "198.51.100.9",
		},
		{
			nama:     "tanpa header sama sekali",
			remote:   "10.0.0.1:5000",
			headers:  nil,
			expected: "10.0.0.1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			got := resolver.clientAddr(req)
			if !got.IsValid() {
				t.Fatalf("alamat tidak valid untuk kasus %q", tc.nama)
			}
			if got.String() != tc.expected {
				t.Fatalf("kasus %q: harus %s, dapat %s", tc.nama, tc.expected, got)
			}
		})
	}
}

func TestClientAddrIPv6HarusDidaftarkanSendiri(t *testing.T) {
	cases := []struct {
		nama     string
		cidrs    []string
		remote   string
		headers  map[string]string
		expected string
	}{
		{
			nama:     "prefix IPv4 saja tidak menutup loopback IPv6, header diabaikan",
			cidrs:    []string{"127.0.0.1/32"},
			remote:   "[::1]:5000",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.7"},
			expected: "::1",
		},
		{
			nama:     "loopback IPv6 terdaftar, XFF dihormati",
			cidrs:    []string{"127.0.0.1/32", "::1/128"},
			remote:   "[::1]:5000",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.7"},
			expected: "203.0.113.7",
		},
		{
			nama:     "loopback IPv6 terdaftar tanpa header, remote dipakai",
			cidrs:    []string{"127.0.0.1/32", "::1/128"},
			remote:   "[::1]:5000",
			headers:  nil,
			expected: "::1",
		},
		{
			nama:     "loopback IPv6 terdaftar, X-Real-IP dipakai saat XFF kosong",
			cidrs:    []string{"127.0.0.1/32", "::1/128"},
			remote:   "[::1]:5000",
			headers:  map[string]string{"X-Real-IP": "203.0.113.7"},
			expected: "203.0.113.7",
		},
		{
			nama:     "IPv4 loopback tetap dihormati meski IPv6 terdaftar",
			cidrs:    []string{"127.0.0.1/32", "::1/128"},
			remote:   "127.0.0.1:5000",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.7"},
			expected: "203.0.113.7",
		},
		{
			nama:     "IPv4-mapped IPv6 dilepas ke IPv4 lalu dicocokkan",
			cidrs:    []string{"10.0.0.0/8"},
			remote:   "[::ffff:10.0.0.1]:5000",
			headers:  map[string]string{"X-Forwarded-For": "203.0.113.7"},
			expected: "203.0.113.7",
		},
	}

	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			resolver := mustResolverBanyak(t, tc.cidrs...)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remote
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}

			got := resolver.clientAddr(req)
			if !got.IsValid() {
				t.Fatalf("alamat tidak valid untuk kasus %q", tc.nama)
			}
			if got.String() != tc.expected {
				t.Fatalf("kasus %q: harus %s, dapat %s", tc.nama, tc.expected, got)
			}
		})
	}
}

func TestIsLocalLoopbackIPv6TetapLokal(t *testing.T) {
	resolver := mustResolver(t, "127.0.0.1/32")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[::1]:5000"
	if !resolver.IsLocal(req) {
		t.Fatal("loopback IPv6 harus tetap dianggap lokal meski tidak terdaftar sebagai proxy")
	}
}

func TestIsLocalTidakTerdapatDiJaringanLokal(t *testing.T) {
	resolver := mustResolver(t, "10.0.0.0/8")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:5000"
	if resolver.IsLocal(req) {
		t.Fatal("alamat publik tidak boleh dianggap lokal")
	}
}
