package repository

import (
	"fmt"
	"strconv"
	"strings"
)

func buildListQueryPG(cols []string, from, whereClause string, args []any, argID int, useTglLeading bool, limitVal int) (string, []any) {
	if useTglLeading {
		subQ := `SELECT kode FROM ` + from + whereClause + fmt.Sprintf(` ORDER BY tgl_entri DESC, kode DESC LIMIT $%d`, argID)
		args = append(args, limitVal)
		argID++
		query := `SELECT ` + qualifyCols(cols, "i") + ` FROM ` + from + ` i JOIN (` + subQ + `) s ON i.kode=s.kode ORDER BY i.kode DESC LIMIT $` + strconv.Itoa(argID)
		args = append(args, limitVal)
		return query, args
	}
	query := `SELECT ` + strings.Join(cols, ", ") + ` FROM ` + from + whereClause + fmt.Sprintf(` ORDER BY kode DESC LIMIT $%d`, argID)
	args = append(args, limitVal)
	return query, args
}
