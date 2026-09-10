package database

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

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

func (r *DBRegistry) Postgres(tenantName string) *pgxpool.Pool {
	r.mu.RLock()
	tenant, exists := r.tenants[tenantName]
	r.mu.RUnlock()
	if !exists {
		panic(fmt.Sprintf("koneksi postgres untuk tenant %s tidak ditemukan", tenantName))
	}
	return tenant.Postgres
}

func (r *DBRegistry) MSSQL(tenantName string) *sql.DB {
	r.mu.RLock()
	tenant, exists := r.tenants[tenantName]
	r.mu.RUnlock()
	if !exists {
		panic(fmt.Sprintf("koneksi mssql untuk tenant %s tidak ditemukan", tenantName))
	}
	return tenant.MSSQL
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
