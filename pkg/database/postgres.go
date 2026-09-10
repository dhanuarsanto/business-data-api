package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
)

type SlogLogger struct{}

func (l *SlogLogger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	if msg == "Query" {
		execTime, ok := data["time"].(time.Duration)
		ms := int64(0)
		if ok {
			ms = execTime.Milliseconds()
		}

		if ms > 500 {
			slog.Warn("Slow SQL Query", "db", "postgres", "sql", data["sql"], "args", data["args"], "duration_ms", ms)
		} else {
			slog.Info("SQL Query Executed", "db", "postgres", "sql", data["sql"], "args", data["args"], "duration_ms", ms)
		}
	}
}

func NewPostgresPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("gagal mem-parsing URL Postgres: %w", err)
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
