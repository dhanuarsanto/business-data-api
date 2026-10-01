package domain

import "errors"

const (
	SourcePostgres = "postgres"
	SourceMSSQL    = "mssql"
)

var ErrSourceNotValid = errors.New("sumber database tidak valid")
