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

func prependBound(whereClause, cond string) string {
	return whereBase + cond + strings.TrimPrefix(whereClause, whereBase)
}
