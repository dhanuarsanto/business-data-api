package http

import (
	"net/url"
	"strconv"
	"time"

	"go.internal/business-data-api/internal/domain"
)

const dateLayout = "2006-01-02"

type pageParams struct {
	PageSize   int
	Cursor     int64
	LimitTotal *int
	StartDate  *time.Time
	EndDate    *time.Time
}

func parsePageParams(q url.Values) pageParams {
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if pageSize <= 0 {
		pageSize = domain.DefaultPageSize
	}
	if pageSize > domain.MaxPageSize {
		pageSize = domain.MaxPageSize
	}

	var cursor int64
	if v, err := strconv.ParseInt(q.Get("cursor"), 10, 64); err == nil {
		cursor = v
	}

	var limitTotal *int
	if val := q.Get("limit"); val != "" {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			if v > domain.MaxLimitTotal {
				v = domain.MaxLimitTotal
			}
			limitTotal = &v
		}
	}

	var startDate, endDate *time.Time
	if val := q.Get("startDate"); val != "" {
		if t, err := time.Parse(dateLayout, val); err == nil {
			startDate = &t
		}
	}
	if val := q.Get("endDate"); val != "" {
		if t, err := time.Parse(dateLayout, val); err == nil {
			t = t.AddDate(0, 0, 1).Add(-time.Second)
			endDate = &t
		}
	}

	return pageParams{
		PageSize:   pageSize,
		Cursor:     cursor,
		LimitTotal: limitTotal,
		StartDate:  startDate,
		EndDate:    endDate,
	}
}
