package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"
)

func TestTrustedProxyResolver(t *testing.T) {
	empty := NewTrustedProxyResolver(nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.5:12345"
	req.Header.Set("X-Forwarded-For", "127.0.0.1")

	if empty.IsLocal(req) {
		t.Fatal("header spoof dari IP publik harus ditolak saat tanpa proxy trusted")
	}

	trusted := NewTrustedProxyResolver([]netip.Prefix{netip.MustParsePrefix("203.0.113.0/24")})
	if !trusted.IsLocal(req) {
		t.Fatal("header dari proxy trusted harus dihormati")
	}

	reqLoop := httptest.NewRequest(http.MethodGet, "/", nil)
	reqLoop.RemoteAddr = "127.0.0.1:12345"
	if !empty.IsLocal(reqLoop) {
		t.Fatal("loopback langsung harus dianggap lokal")
	}
}

func TestRequestBodyLimit(t *testing.T) {
	logger.SetupLogger("test")

	handler := RequestBodyLimit(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	big := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", 4096)))
	big.Header.Set("Content-Length", "4096")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, big)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("body > batas harus 413, dapat %d", rec.Code)
	}

	small := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("ok"))
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, small)
	if rec2.Code != http.StatusOK {
		t.Fatalf("body normal harus lolos, dapat %d", rec2.Code)
	}
}

func TestRequireTokenTenantMatch(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")
	jwt.InitJWT("rahasia-uji-128bit", 1*time.Hour, "business-data-api")

	token, err := jwt.GenerateToken(1, "budi", "sa", "maxtop")
	if err != nil {
		t.Fatalf("generate token gagal: %v", err)
	}

	run := func(tenant string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/"+tenant+"/inbox", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenant", tenant)
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		RequireToken()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)
		return rec.Code
	}

	if code := run("maxtop"); code != http.StatusOK {
		t.Fatalf("token tenant sama harus lolos, dapat %d", code)
	}
	if code := run("toplink"); code != http.StatusForbidden {
		t.Fatalf("token tenant beda harus 403, dapat %d", code)
	}
}

func TestRateLimiterFirstRequests(t *testing.T) {
	rl := NewRateLimiter(0, 2)

	if !rl.allow("1.1.1.1") {
		t.Fatal("request pertama harus lolos (kapasitas penuh)")
	}
	if !rl.allow("1.1.1.1") {
		t.Fatal("request kedua harus lolos")
	}
	if rl.allow("1.1.1.1") {
		t.Fatal("request ketiga harus ditolak")
	}
}

func TestRateLimiterStop(t *testing.T) {
	rl := NewRateLimiter(10, 5)
	rl.Stop()
	rl.Stop()
}

func TestPostgresWriteGuard(t *testing.T) {
	guardOn := PostgresWriteGuard(true)
	guardOff := PostgresWriteGuard(false)

	run := func(guard func(http.Handler) http.Handler, method, source string) int {
		req := httptest.NewRequest(method, "/", nil)
		if source != "" {
			req.Header.Set("X-DB-Source", source)
		}
		rec := httptest.NewRecorder()
		guard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)
		return rec.Code
	}

	if c := run(guardOff, http.MethodPost, "postgres"); c != http.StatusForbidden {
		t.Fatalf("POST postgres saat guard mati harus 403, dapat %d", c)
	}
	if c := run(guardOff, http.MethodPut, "postgres"); c != http.StatusForbidden {
		t.Fatalf("PUT postgres saat guard mati harus 403, dapat %d", c)
	}
	if c := run(guardOff, http.MethodPost, "mssql"); c != http.StatusOK {
		t.Fatalf("POST mssql harus lolos, dapat %d", c)
	}
	if c := run(guardOff, http.MethodGet, "postgres"); c != http.StatusOK {
		t.Fatalf("GET harus lolos, dapat %d", c)
	}
	if c := run(guardOn, http.MethodPost, "postgres"); c != http.StatusOK {
		t.Fatalf("POST postgres saat guard nyala harus lolos, dapat %d", c)
	}
}

func TestPanicRecoverer(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("development")

	panicHandler := PanicRecoverer()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("simulasi crash internal")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	panicHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic harus result 500, dapat %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Terjadi kesalahan sistem yang fatal") {
		t.Fatalf("response harus berisi pesan 500, dapat: %s", rec.Body.String())
	}
}

func TestSecurityTracer(t *testing.T) {
	logger.SetupLogger("test")

	var capturedTraceID string
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tCtx := logger.GetTraceContext(r.Context())
		capturedTraceID = tCtx.TraceID
		w.WriteHeader(http.StatusOK)
	})

	handler := SecurityTracer()(inner)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	handler.ServeHTTP(rec, req)

	if capturedTraceID == "" || capturedTraceID == "UNKNOWN" {
		t.Fatalf("trace ID harus terbentuk, dapat: %q", capturedTraceID)
	}
	if !strings.HasPrefix(capturedTraceID, "TRC-") {
		t.Fatalf("trace ID harus diawali TRC-, dapat: %q", capturedTraceID)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("harus 200, dapat %d", rec.Code)
	}
}

func TestRequireRole(t *testing.T) {
	logger.SetupLogger("test")
	response.Init("test")
	jwt.InitJWT("rahasia-uji-128bit", 1*time.Hour, "business-data-api")

	tokenOK, _ := jwt.GenerateToken(1, "budi", "sa", "maxtop")
	tokenUnknown, _ := jwt.GenerateToken(1, "hacker", "unknown_role", "maxtop")

	run := func(token string) int {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/maxtop/inbox", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("tenant", "maxtop")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler := RequireToken()(RequireRole("sa", "op")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := run(tokenOK); code != http.StatusOK {
		t.Fatalf("role 'sa' harus lolos role 'sa,op', dapat %d", code)
	}
	if code := run(tokenUnknown); code != http.StatusForbidden {
		t.Fatalf("role 'unknown_role' harus 403, dapat %d", code)
	}
}