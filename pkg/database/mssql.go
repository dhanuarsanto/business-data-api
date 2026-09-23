package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"log/slog"
	"time"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/ngrok/sqlmw"
	"go.internal/business-data-api/pkg/logger"
)

type mssqlLogger struct {
	sqlmw.NullInterceptor
}

func logMSSQLQuery(ctx context.Context, query string, start time.Time) {
	ms := time.Since(start).Milliseconds()
	tCtx := logger.GetTraceContext(ctx)

	if ms > 500 {
		slog.Warn("Slow SQL Query", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "path", tCtx.Path, "db", "mssql", "sql", query, "duration_ms", ms)
	} else {
		slog.Info("SQL Query Executed", "trace_id", tCtx.TraceID, "developer", tCtx.Developer, "path", tCtx.Path, "db", "mssql", "sql", query, "duration_ms", ms)
	}
}

func (l *mssqlLogger) ConnQueryContext(ctx context.Context, conn driver.QueryerContext, query string, args []driver.NamedValue) (context.Context, driver.Rows, error) {
	start := time.Now()
	rows, err := conn.QueryContext(ctx, query, args)
	logMSSQLQuery(ctx, query, start)
	return ctx, rows, err
}

func (l *mssqlLogger) ConnExecContext(ctx context.Context, conn driver.ExecerContext, query string, args []driver.NamedValue) (driver.Result, error) {
	start := time.Now()
	res, err := conn.ExecContext(ctx, query, args)
	logMSSQLQuery(ctx, query, start)
	return res, err
}

func init() {
	sql.Register("mssql-logged", sqlmw.Driver(&mssql.Driver{}, new(mssqlLogger)))
}

func NewMSSQLDB(ctx context.Context, connString string) (*sql.DB, error) {
	db, err := sql.Open("mssql-logged", connString)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi MSSQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("MSSQL tidak dapat di-ping: %w", err)
	}

	return db, nil
}
