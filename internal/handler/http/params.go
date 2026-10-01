package http

import (
	"net/url"
	"strconv"
	"time"

	"go.internal/business-data-api/internal/domain"
)

const dateLayout = "2006-01-02"

type listParams struct {
	Limit     int
	StartDate *time.Time
	EndDate   *time.Time
}

func parseListParams(q url.Values) listParams {
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 {
		limit = domain.DefaultLimit
	}
	if limit > domain.MaxLimit {
		limit = domain.MaxLimit
	}

	var startDate, endDate *time.Time
	if val := q.Get("startDate"); val != "" {
		if t, err := time.Parse(dateLayout, val); err == nil {
			startDate = &t
		}
	}
	if val := q.Get("endDate"); val != "" {
		if t, err := time.Parse(dateLayout, val); err == nil {
			akhir := t.AddDate(0, 0, 1).Add(-time.Second)
			endDate = &akhir
		}
	}

	return listParams{
		Limit:     limit,
		StartDate: startDate,
		EndDate:   endDate,
	}
}
