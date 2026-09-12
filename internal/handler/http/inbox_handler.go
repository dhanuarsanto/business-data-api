package http

import (
	"encoding/json"
	"net/http"
	"strconv"
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

type InboxHandler struct {
	usecase usecase.InboxUsecase
}

func (h *InboxHandler) GetInbox(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = "postgres"
	}

	queryParams := r.URL.Query()

	limit, _ := strconv.Atoi(queryParams.Get("limit"))
	cursor, _ := strconv.ParseInt(queryParams.Get("cursor"), 10, 64)

	var startDatePtr, endDatePtr *time.Time
	if val := queryParams.Get("startDate"); val != "" {
		if t, err := time.Parse("2006-01-02", val); err == nil {
			startDatePtr = &t
		}
	}
	if val := queryParams.Get("endDate"); val != "" {
		if t, err := time.Parse("2006-01-02", val); err == nil {
			endDatePtr = &t
		}
	}

	var terminalPtr *int
	if val := queryParams.Get("terminal"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			terminalPtr = &v
		}
	}

	var resellerPtr *string
	if val := queryParams.Get("reseller"); val != "" {
		resellerPtr = &val
	}

	var pengirimPtr *string
	if val := queryParams.Get("pengirim"); val != "" {
		pengirimPtr = &val
	}

	var tipePtr *string
	if val := queryParams.Get("tipe"); val != "" {
		tipePtr = &val
	}

	var statusPtr *int
	if val := queryParams.Get("status"); val != "" {
		if s, err := strconv.Atoi(val); err == nil {
			statusPtr = &s
		}
	}

	var reqFromResellerPtr *bool
	if val := queryParams.Get("requestFromReseller"); val != "" {
		b := val == "true" || val == "1"
		reqFromResellerPtr = &b
	}

	var jawFromProviderPtr *bool
	if val := queryParams.Get("jawabanFromProvider"); val != "" {
		b := val == "true" || val == "1"
		jawFromProviderPtr = &b
	}

	filter := domain.InboxFilter{
		StartDate:           startDatePtr,
		EndDate:             endDatePtr,
		Limit:               limit,
		Terminal:            terminalPtr,
		Reseller:            resellerPtr,
		Pengirim:            pengirimPtr,
		Tipe:                tipePtr,
		Status:              statusPtr,
		Pesan:               queryParams.Get("pesan"),
		RequestFromReseller: reqFromResellerPtr,
		JawabanFromProvider: jawFromProviderPtr,
		Search:              queryParams.Get("search"),
		Cursor:              cursor,
	}

	data, totalData, hasNextPage, err := h.usecase.GetInbox(r.Context(), tenant, dbSource, filter)
	if err != nil {
		response.Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	var nextCursor int64
	if len(data) > 0 {
		nextCursor = data[len(data)-1].Kode
	}

	response.Success(w, r, map[string]any{
		"trace_id": tCtx.TraceID,
		"data":     data,
		"meta": response.CursorPaginationMeta{
			TotalData:   totalData,
			HasNextPage: hasNextPage,
			HasPrevPage: filter.Cursor > 0,
			NextCursor:  nextCursor,
		},
	})
}

func (h *InboxHandler) InsertInbox(w http.ResponseWriter, r *http.Request) {
	tCtx := logger.GetTraceContext(r.Context())
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = "postgres"
	}

	var payload dto.InsertInboxRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	tipePengirim := payload.TipePengirim
	if tipePengirim == "" {
		tipePengirim = "S"
	}
	status := 0
	if payload.Status != nil {
		status = *payload.Status
	}
	isJawaban := 0
	if payload.IsJawaban != nil {
		isJawaban = *payload.IsJawaban
	}

	data := domain.Inbox{
		Pesan:         payload.Pesan,
		Pengirim:      payload.Pengirim,
		Penerima:      payload.Penerima,
		TipePengirim:  tipePengirim,
		Status:        status,
		KodeTerminal:  payload.KodeTerminal,
		KodeReseller:  payload.KodeReseller,
		KodeTransaksi: payload.KodeTransaksi,
		ServiceCenter: payload.ServiceCenter,
		IsJawaban:     isJawaban,
		IsCs:          payload.IsCs,
		KodeJawabanCs: payload.KodeJawabanCs,
		Hash:          payload.Hash,
	}

	if err := h.usecase.InsertInbox(r.Context(), tenant, dbSource, data); err != nil {
		response.Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(w, r, map[string]any{"trace_id": tCtx.TraceID, "message": "Sukses"})
}

func (h *InboxHandler) UpdateInbox(w http.ResponseWriter, r *http.Request) {
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
		dbSource = "postgres"
	}

	var payload dto.UpdateInboxRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	if err := h.usecase.UpdateInbox(r.Context(), tenant, dbSource, kode, payload); err != nil {
		response.Error(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	response.Success(w, r, map[string]any{"trace_id": tCtx.TraceID, "message": "Sukses"})
}

func NewInboxHandler(usecase usecase.InboxUsecase) *InboxHandler {
	return &InboxHandler{usecase: usecase}
}

func (h *InboxHandler) RegisterRoutes(r *chi.Mux, cfg *config.Config, roleMatrix map[string][]string) {
	r.Route("/api/v1/{tenant}/inbox", func(inbox chi.Router) {
		inbox.Use(api_middleware.RequireToken())
		inbox.Group(func(read chi.Router) {
			read.Use(api_middleware.RequireRole(roleMatrix["ReadInbox"]...))
			read.Get("/", h.GetInbox)
		})

		inbox.Group(func(write chi.Router) {
			write.Use(api_middleware.RequireRole(roleMatrix["WriteInbox"]...))
			write.Use(api_middleware.PostgresWriteGuard(cfg.PostgresWriteEnabled))
			write.Post("/", h.InsertInbox)
			write.Put("/{kode}", h.UpdateInbox)
		})
	})
}
