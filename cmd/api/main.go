package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/pkg/database"
	"go.internal/business-data-api/pkg/jwt"
	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"

	http_handler "go.internal/business-data-api/internal/handler/http"
	api_middleware "go.internal/business-data-api/internal/middleware"
)

func main() {
	cfg := config.LoadConfig()
	if len(cfg.JWTSecret) < jwt.MinSecretLength {
		slog.Error("JWT_SECRET terlalu pendek — wajib minimal 32 byte demi keamanan token")
		os.Exit(1)
	}
	logger.SetupLogger(cfg.AppEnv)
	response.Init(cfg.AppEnv)
	jwt.InitJWT(cfg.JWTSecret, cfg.JWTTokenDuration, cfg.JWTIssuer)

	slog.Info("Menjalankan API", "mode", cfg.AppEnv, "port", cfg.Port)

	dbRegistry := database.NewDBRegistry()
	ctx := context.Background()

	pgMaxtop, err := database.NewPostgresPool(ctx, cfg.PostgresMaxtopURL, pgPoolOptions(cfg))
	if err != nil {
		slog.Error("Gagal koneksi Postgres Maxtop", "error", err)
		os.Exit(1)
	}
	msMaxtop, err := database.NewMSSQLDB(ctx, cfg.MSSQLMaxtopURL, msPoolOptions(cfg))
	if err != nil {
		slog.Error("Gagal koneksi MSSQL Maxtop", "error", err)
		os.Exit(1)
	}
	dbRegistry.Register(tenantMaxtop, pgMaxtop, msMaxtop)

	pgPandora, err := database.NewPostgresPool(ctx, cfg.PostgresPandoraURL, pgPoolOptions(cfg))
	if err != nil {
		slog.Error("Gagal koneksi Postgres Pandora", "error", err)
		os.Exit(1)
	}
	msPandora, err := database.NewMSSQLDB(ctx, cfg.MSSQLPandoraURL, msPoolOptions(cfg))
	if err != nil {
		slog.Error("Gagal koneksi MSSQL Pandora", "error", err)
		os.Exit(1)
	}
	dbRegistry.Register(tenantPandora, pgPandora, msPandora)

	pgToplink, err := database.NewPostgresPool(ctx, cfg.PostgresToplinkURL, pgPoolOptions(cfg))
	if err != nil {
		slog.Error("Gagal koneksi Postgres Toplink", "error", err)
		os.Exit(1)
	}
	msToplink, err := database.NewMSSQLDB(ctx, cfg.MSSQLToplinkURL, msPoolOptions(cfg))
	if err != nil {
		slog.Error("Gagal koneksi MSSQL Toplink", "error", err)
		os.Exit(1)
	}
	dbRegistry.Register(tenantToplink, pgToplink, msToplink)

	trustedProxies, err := cfg.ParseTrustedProxies()
	if err != nil {
		slog.Error("Konfigurasi TRUSTED_PROXIES tidak valid", "error", err)
		os.Exit(1)
	}
	if len(trustedProxies) == 0 {
		if cfg.AppEnv == "production" && cfg.GlobalLocalOnly && !cfg.AllowDirectClients {
			slog.Error("Konfigurasi tidak aman: GLOBAL_LOCAL_ONLY=true tetapi jalur klien belum dideklarasikan. Isi TRUSTED_PROXIES (IP/CIDR proxy/nginx/LB) bila API di belakang reverse-proxy, atau set ALLOW_DIRECT_CLIENTS=true bila klien terhubung langsung ke API.")
			os.Exit(1)
		}
		slog.Warn("TRUSTED_PROXIES kosong — rate limit memakai RemoteAddr. Set IP/CIDR proxy (nginx/IIS/LB) bila API di belakang reverse-proxy")
	}
	ipResolver := api_middleware.NewTrustedProxyResolver(trustedProxies)

	activeModules, networkMatrix, roleMatrix := BuildModules(dbRegistry, cfg, ipResolver)
	r, limiters, keyManager := http_handler.SetupRoutes(cfg, ipResolver, networkMatrix, roleMatrix, activeModules...)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadHeaderTimeout: cfg.ServerReadHeaderTimeout,
		ReadTimeout:       cfg.ServerReadTimeout,
		WriteTimeout:      cfg.ServerWriteTimeout,
		IdleTimeout:       cfg.ServerIdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Kesalahan server", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("Sinyal berhenti diterima, mematikan server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ServerShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server dipaksa mati", "error", err)
	}

	for _, l := range limiters {
		l.Stop()
	}
	keyManager.Stop()

	dbRegistry.CloseAll()
	slog.Info("Server berhasil dimatikan dengan aman")
}

func pgPoolOptions(cfg *config.Config) database.PostgresPoolOptions {
	return database.PostgresPoolOptions{
		MaxConns:          cfg.PostgresMaxConns,
		MinConns:          cfg.PostgresMinConns,
		MaxConnIdleTime:   cfg.PostgresMaxConnIdleTime,
		MaxConnLifetime:   cfg.PostgresMaxConnLifetime,
		HealthCheckPeriod: cfg.PostgresHealthCheckPeriod,
	}
}

func msPoolOptions(cfg *config.Config) database.MSSQLPoolOptions {
	return database.MSSQLPoolOptions{
		MaxOpenConns:    cfg.MSSQLMaxOpenConns,
		MaxIdleConns:    cfg.MSSQLMaxIdleConns,
		MaxConnLifetime: cfg.MSSQLMaxConnLifetime,
	}
}
