package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/internal/usecase"
	"go.internal/business-data-api/pkg/logger"
	"go.internal/business-data-api/pkg/response"

	api_middleware "go.internal/business-data-api/internal/middleware"
)

type OutboxHandler struct {
	usecase *usecase.OutboxUsecase
}

func (h *OutboxHandler) GetOutbox(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = domain.SourcePostgres
	}

	queryParams := r.URL.Query()

	pageSize, _ := strconv.Atoi(queryParams.Get("pageSize"))
	if pageSize <= 0 {
		pageSize = domain.DefaultPageSize
	}
	if pageSize > domain.MaxPageSize {
		pageSize = domain.MaxPageSize
	}
	cursor, _ := strconv.ParseInt(queryParams.Get("cursor"), 10, 64)

	var limitTotalPtr *int
	if val := queryParams.Get("limit"); val != "" {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			if v > domain.MaxLimitTotal {
				v = domain.MaxLimitTotal
			}
			limitTotalPtr = &v
		}
	}

	var startDatePtr, endDatePtr *time.Time
	if val := queryParams.Get("startDate"); val != "" {
		if t, err := time.Parse("2006-01-02", val); err == nil {
			startDatePtr = &t
		}
	}
	if val := queryParams.Get("endDate"); val != "" {
		if t, err := time.Parse("2006-01-02", val); err == nil {
			t = t.AddDate(0, 0, 1).Add(-time.Second)
			endDatePtr = &t
		}
	}

	var resellerPtr *string
	if val := queryParams.Get("reseller"); val != "" {
		v := strings.TrimSpace(val)
		resellerPtr = &v
	}

	var penerimaPtr *string
	if val := queryParams.Get("penerima"); val != "" {
		v := strings.TrimSpace(val)
		penerimaPtr = &v
	}

	var tipePtr *string
	if val := queryParams.Get("tipe"); val != "" {
		v := strings.TrimSpace(val)
		tipePtr = &v
	}

	var statusPtr *int16
	if statusStr := queryParams.Get("status"); statusStr != "" {
		if s, err := strconv.ParseInt(statusStr, 10, 16); err == nil {
			val := int16(s)
			statusPtr = &val
		}
	}

	var replyToResellerPtr *bool
	if val := queryParams.Get("replyToReseller"); val != "" {
		b := val == "true" || val == "1"
		replyToResellerPtr = &b
	}

	var perintahProviderPtr *bool
	if val := queryParams.Get("perintahProvider"); val != "" {
		b := val == "true" || val == "1"
		perintahProviderPtr = &b
	}

	filter := domain.OutboxFilter{
		StartDate:        startDatePtr,
		EndDate:          endDatePtr,
		PageSize:         pageSize,
		LimitTotal:       limitTotalPtr,
		Reseller:         resellerPtr,
		Penerima:         penerimaPtr,
		Tipe:             tipePtr,
		Status:           statusPtr,
		Pesan:            strings.TrimSpace(queryParams.Get("pesan")),
		ReplyToReseller:  replyToResellerPtr,
		PerintahProvider: perintahProviderPtr,
		Cursor:           cursor,
	}

	data, hasNextPage, err := h.usecase.GetOutbox(r.Context(), tenant, dbSource, filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var nextCursor int64
	if len(data) > 0 {
		nextCursor = data[len(data)-1].Kode
	}

	response.Success(w, r, map[string]any{
		"trace_id": tCtx.TraceID,
		"items":    data,
		"meta": response.CursorPaginationMeta{
			HasNextPage: hasNextPage,
			HasPrevPage: filter.Cursor > 0,
			NextCursor:  nextCursor,
		},
	})
}

func (h *OutboxHandler) CreateOutbox(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = domain.SourcePostgres
	}

	var payload dto.CreateOutboxRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	data := domain.Outbox{
		Penerima:      payload.Penerima,
		TipePenerima:  payload.TipePenerima,
		Pesan:         payload.Pesan,
		Status:        payload.Status,
		BebasBiaya:    payload.BebasBiaya,
		KodeInbox:     payload.KodeInbox,
		KodeTransaksi: payload.KodeTransaksi,
		KodeReseller:  payload.KodeReseller,
		IsPerintah:    payload.IsPerintah,
		KodeModul:     payload.KodeModul,
		Prioritas:     payload.Prioritas,
		ModulProses:   payload.ModulProses,
		Pengirim:      payload.Pengirim,
		KodeTerminal:  payload.KodeTerminal,
		CtrKirim:      payload.CtrKirim,
	}

	if err := h.usecase.CreateOutbox(r.Context(), tenant, dbSource, data); err != nil {
		writeError(w, r, err)
		return
	}
	response.SuccessCreated(w, r, map[string]any{"trace_id": tCtx.TraceID, "message": "Sukses"})
}

func (h *OutboxHandler) UpdateOutbox(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	kodeStr := chi.URLParam(r, "kode")
	kode, err := strconv.ParseInt(kodeStr, 10, 64)
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, "Parameter kode harus berupa angka")
		return
	}

	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = domain.SourcePostgres
	}

	var payload dto.UpdateOutboxRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	if err := h.usecase.UpdateOutbox(r.Context(), tenant, dbSource, kode, payload); err != nil {
		writeError(w, r, err)
		return
	}
	response.Success(w, r, map[string]any{"trace_id": tCtx.TraceID, "message": "Sukses"})
}

func NewOutboxHandler(usecase *usecase.OutboxUsecase) *OutboxHandler {
	return &OutboxHandler{usecase: usecase}
}

func (h *OutboxHandler) RegisterRoutes(_public, protected chi.Router, cfg *config.Config, roleMatrix map[string][]string) {
	protected.Route("/api/v1/{tenant}/outbox", func(outbox chi.Router) {
		outbox.Group(func(read chi.Router) {
			read.Use(api_middleware.RequireRole(roleMatrix["ReadOutbox"]...))
			read.Get("/", h.GetOutbox)
		})

		outbox.Group(func(write chi.Router) {
			write.Use(api_middleware.RequireRole(roleMatrix["WriteOutbox"]...))
			write.Use(api_middleware.PostgresWriteGuard(cfg.PostgresWriteEnabled))
			write.Post("/", h.CreateOutbox)
			write.Put("/{kode}", h.UpdateOutbox)
		})
	})
}
