package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"go.internal/business-data-api/pkg/logger"
)

type SlogLogger struct{}

func (l *SlogLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	if msg == "Query" {
		execTime, ok := data["time"].(time.Duration)
		ms := int64(0)
		if ok {
			ms = execTime.Milliseconds()
		}

		tCtx := logger.GetTraceContext(ctx)

		if execTime > slowQueryThreshold {
			slog.Warn("Slow SQL Query", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "path", tCtx.Path, "db", "postgres", "sql", data["sql"], "duration_ms", ms)
		} else {
			slog.Info("SQL Query Executed", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "path", tCtx.Path, "db", "postgres", "sql", data["sql"], "duration_ms", ms)
		}
	}
}

type PostgresPoolOptions struct {
	MaxConns          int
	MinConns          int
	MaxConnIdleTime   time.Duration
	MaxConnLifetime   time.Duration
	HealthCheckPeriod time.Duration
}

func NewPostgresPool(ctx context.Context, connString string, opts PostgresPoolOptions) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("gagal mem-parsing URL Postgres: %w", err)
	}

	if opts.MaxConns > 0 {
		poolConfig.MaxConns = int32(opts.MaxConns)
	}
	if opts.MinConns > 0 {
		poolConfig.MinConns = int32(opts.MinConns)
	}
	if opts.MaxConnIdleTime > 0 {
		poolConfig.MaxConnIdleTime = opts.MaxConnIdleTime
	}
	if opts.MaxConnLifetime > 0 {
		poolConfig.MaxConnLifetime = opts.MaxConnLifetime
	}
	if opts.HealthCheckPeriod > 0 {
		poolConfig.HealthCheckPeriod = opts.HealthCheckPeriod
	}

	poolConfig.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   &SlogLogger{},
		LogLevel: tracelog.LogLevelInfo,
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat connection pool Postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Postgres tidak dapat di-ping: %w", err)
	}

	return pool, nil
}
