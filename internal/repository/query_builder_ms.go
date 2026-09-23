package repository

import (
	"database/sql"
	"strings"
)

func buildPageQueryMS(cols, from, whereClause string, namedArgs []any, useTglLeading bool, limitVal int) (string, []any) {
	if useTglLeading {
		subQ := `SELECT TOP (@p_limit) kode FROM ` + from + whereClause + ` ORDER BY tgl_entri DESC, kode DESC`
		namedArgs = append(namedArgs, sql.Named("p_limit", limitVal))
		query := `SELECT i.` + strings.ReplaceAll(cols, ", ", ", i.") + ` FROM ` + from + ` i JOIN (` + subQ + `) s ON i.kode=s.kode ORDER BY i.kode DESC`
		return query, namedArgs
	}
	query := `SELECT TOP (@p_limit) ` + cols + ` FROM ` + from + whereClause + ` ORDER BY kode DESC`
	namedArgs = append(namedArgs, sql.Named("p_limit", limitVal))
	return query, namedArgs
}