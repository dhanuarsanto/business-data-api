package http

import (
	"errors"
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

var (
	ErrInvalidStartDate = errors.New("startDate format invalid, expected YYYY-MM-DD")
	ErrInvalidEndDate   = errors.New("endDate format invalid, expected YYYY-MM-DD")
	ErrInvalidLimit     = errors.New("limit must be positive integer")
)

func parseListParams(q url.Values) (listParams, error) {
	limitStr := q.Get("limit")
	var limit int
	if limitStr == "" {
		limit = domain.DefaultLimit
	} else {
		l, err := strconv.Atoi(limitStr)
		if err != nil || l < 0 {
			return listParams{}, ErrInvalidLimit
		}
		if l == 0 {
			limit = domain.DefaultLimit
		} else if l > domain.MaxLimit {
			limit = domain.MaxLimit
		} else {
			limit = l
		}
	}

	var startDate, endDate *time.Time
	if val := q.Get("startDate"); val != "" {
		if val == "" {
			return listParams{}, ErrInvalidStartDate
		}
		t, err := time.Parse(dateLayout, val)
		if err != nil {
			return listParams{}, ErrInvalidStartDate
		}
		startDate = &t
	}
	if val := q.Get("endDate"); val != "" {
		if val == "" {
			return listParams{}, ErrInvalidEndDate
		}
		t, err := time.Parse(dateLayout, val)
		if err != nil {
			return listParams{}, ErrInvalidEndDate
		}
		akhir := t.AddDate(0, 0, 1).Add(-time.Second)
		endDate = &akhir
	}

	return listParams{
		Limit:     limit,
		StartDate: startDate,
		EndDate:   endDate,
	}, nil
}
