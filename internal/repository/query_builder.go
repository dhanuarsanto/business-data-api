package repository

import "strings"

const whereBase = " WHERE 1=1"

func qualifyCols(cols []string, alias string) string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = alias + "." + c
	}
	return strings.Join(out, ", ")
}

// prependBound menyisipkan syarat batas di depan WHERE yang sudah dibangun filter builder.
// Semua filter builder wajib diawali whereBase; diuji di filter_test.go.
func prependBound(whereClause, cond string) string {
	return whereBase + cond + strings.TrimPrefix(whereClause, whereBase)
}
