package database

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrTenantNotFound = errors.New("tenant tidak ditemukan")

type TenantDB struct {
	Postgres *pgxpool.Pool
	MSSQL    *sql.DB
}

type DBRegistry struct {
	mu      sync.RWMutex
	tenants map[string]*TenantDB
}

func NewDBRegistry() *DBRegistry {
	return &DBRegistry{
		tenants: make(map[string]*TenantDB),
	}
}

func (r *DBRegistry) Register(tenantName string, pg *pgxpool.Pool, ms *sql.DB) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tenants[tenantName] = &TenantDB{
		Postgres: pg,
		MSSQL:    ms,
	}
}

func (r *DBRegistry) Postgres(tenantName string) (*pgxpool.Pool, error) {
	r.mu.RLock()
	tenant, exists := r.tenants[tenantName]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTenantNotFound, tenantName)
	}
	return tenant.Postgres, nil
}

func (r *DBRegistry) MSSQL(tenantName string) (*sql.DB, error) {
	r.mu.RLock()
	tenant, exists := r.tenants[tenantName]
	r.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrTenantNotFound, tenantName)
	}
	return tenant.MSSQL, nil
}

func (r *DBRegistry) CloseAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, tenant := range r.tenants {
		if tenant.Postgres != nil {
			tenant.Postgres.Close()
		}
		if tenant.MSSQL != nil {
			tenant.MSSQL.Close()
		}
	}
}
