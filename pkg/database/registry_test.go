package database

import "testing"

func TestRegistryReturnsErrorForUnknownTenant(t *testing.T) {
	r := NewDBRegistry()

	if _, err := r.Postgres("nope"); err == nil {
		t.Fatal("Postgres tenant tak dikenal harus error, bukan panic")
	}
	if _, err := r.MSSQL("nope"); err == nil {
		t.Fatal("MSSQL tenant tak dikenal harus error, bukan panic")
	}
}