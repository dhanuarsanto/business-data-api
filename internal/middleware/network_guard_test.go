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

func TestIsLocalTidakTerdapatDiJaringanLokal(t *testing.T) {
	resolver := mustResolver(t, "10.0.0.0/8")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:5000"
	if resolver.IsLocal(req) {
		t.Fatal("alamat publik tidak boleh dianggap lokal")
	}
}
