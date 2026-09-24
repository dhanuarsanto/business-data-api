package http_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go.internal/business-data-api/internal/config"
	http_handler "go.internal/business-data-api/internal/handler/http"
	api_middleware "go.internal/business-data-api/internal/middleware"
	"go.internal/business-data-api/internal/repository"
	"go.internal/business-data-api/internal/usecase"
	"go.internal/business-data-api/pkg/database"
	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/response"
)

func TestAPIEndToEnd(t *testing.T) {
	dir := t.TempDir()
	oldCwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir gagal: %v", err)
	}
	defer os.Chdir(oldCwd)
	if err := os.WriteFile("api_keys.json", []byte(`{"key_e2e":"E2E Dev"}`), 0o644); err != nil {
		t.Fatalf("tulis api_keys.json gagal: %v", err)
	}

	response.Init("test")
	jwt.InitJWT("rahasia-e2e-uji", 1*time.Hour, "e2e")

	cfg := &config.Config{
		AppEnv:               "test",
		GlobalLocalOnly:      false,
		PostgresWriteEnabled: false,
		MaxBodyBytes:         1048576,
		CookieSecure:         false,
	}
	prefixes, err := cfg.ParseTrustedProxies()
	if err != nil {
		t.Fatalf("parse trusted proxies gagal: %v", err)
	}
	resolver := api_middleware.NewTrustedProxyResolver(prefixes)
	registry := database.NewDBRegistry()

	networkMatrix := map[string]bool{"*": true, "sa": false}
	roleMatrix := map[string][]string{
		"ManageUsers": {"sa"},
		"ReadInbox":   {"sa", "op", "opout"},
		"WriteInbox":  {"sa"},
		"ReadOutbox":  {"sa", "op", "opout"},
		"WriteOutbox": {"sa"},
	}

	userRepos := repository.NewUserRepositories(registry)
	authHandler := http_handler.NewAuthHandler(usecase.NewAuthUsecase(userRepos.PG, userRepos.MS), networkMatrix, resolver, cfg.CookieSecure)
	inboxRepos := repository.NewInboxRepositories(registry)
	inboxHandler := http_handler.NewInboxHandler(usecase.NewInboxUsecase(inboxRepos.PG, inboxRepos.MS))
	outboxRepos := repository.NewOutboxRepositories(registry)
	outboxHandler := http_handler.NewOutboxHandler(usecase.NewOutboxUsecase(outboxRepos.PG, outboxRepos.MS))
	modules := []http_handler.Module{authHandler, inboxHandler, outboxHandler}

	router, limiters, keyManager := http_handler.SetupRoutes(cfg, resolver, networkMatrix, roleMatrix, modules...)
	defer func() {
		for _, l := range limiters {
			l.Stop()
		}
		keyManager.Stop()
		registry.CloseAll()
	}()

	srv := httptest.NewServer(router)
	defer srv.Close()

	token, err := jwt.GenerateToken(1, "budi", "sa", "maxtop")
	if err != nil {
		t.Fatalf("generate token gagal: %v", err)
	}

	do := func(method, path, apiKey, bearer string, body io.Reader) (*http.Response, string) {
		req, _ := http.NewRequest(method, srv.URL+path, body)
		if apiKey != "" {
			req.Header.Set("X-API-KEY", apiKey)
		}
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		resp, err := srv.Client().Do(req)
		if err != nil {
			t.Fatalf("request %s %s gagal: %v", method, path, err)
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(resp.Body)
		return resp, string(raw)
	}

	if resp, _ := do(http.MethodGet, "/health", "", "", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("health tanpa API key harus 401, dapat %d", resp.StatusCode)
	}
	if resp, _ := do(http.MethodGet, "/health", "key_salah", "", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("health dengan API key salah harus 401, dapat %d", resp.StatusCode)
	}
	if resp, _ := do(http.MethodGet, "/health", "key_e2e", "", nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("health dengan API key benar harus 200, dapat %d", resp.StatusCode)
	}

	inboxPath := "/api/v1/maxtop/inbox"
	if resp, _ := do(http.MethodGet, inboxPath, "key_e2e", "", nil); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET inbox tanpa token harus 401, dapat %d", resp.StatusCode)
	}
	if resp, _ := do(http.MethodGet, inboxPath, "key_e2e", token, nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET inbox tenant tak terdaftar harus 404, dapat %d", resp.StatusCode)
	}
	if resp, _ := do(http.MethodPost, inboxPath, "key_e2e", token, strings.NewReader(strings.Repeat("a", 2*1024*1024))); resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("POST inbox body besar harus 413, dapat %d", resp.StatusCode)
	}
	if resp, _ := do(http.MethodPost, "/api/v1/maxtop/auth/login", "key_e2e", "", strings.NewReader(`{"username":"budi","password":"x"}`)); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("login user tak ada harus 401, dapat %d", resp.StatusCode)
	}
}