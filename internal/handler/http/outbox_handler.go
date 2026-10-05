package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.internal/business-data-api/internal/config"
	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/internal/usecase"
	"go.internal/business-data-api/pkg/response"

	api_middleware "go.internal/business-data-api/internal/middleware"
)

type OutboxHandler struct {
	usecase *usecase.OutboxUsecase
}

func (h *OutboxHandler) GetOutbox(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = domain.SourcePostgres
	}

	queryParams := r.URL.Query()
	params, err := parseListParams(queryParams)
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	resellerPtr := parseStringFilter(queryParams.Get("reseller"))
	penerimaPtr := parseStringFilter(queryParams.Get("penerima"))
	tipePtr := parseStringFilter(queryParams.Get("tipe"))

	statusPtr, statusMinPtr, err := parseStatusFilter(queryParams.Get("status"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	replyToResellerPtr, err := parseBoolFilter(queryParams.Get("replyToReseller"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	perintahProviderPtr, err := parseBoolFilter(queryParams.Get("perintahProvider"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	pesan := strings.TrimSpace(queryParams.Get("pesan"))
	if len(pesan) > maxStringLen {
		pesan = ""
	}

	filter := domain.OutboxFilter{
		StartDate:        params.StartDate,
		EndDate:          params.EndDate,
		Limit:            params.Limit,
		Reseller:         resellerPtr,
		Penerima:         penerimaPtr,
		Tipe:             tipePtr,
		Status:           statusPtr,
		StatusMin:        statusMinPtr,
		Pesan:            pesan,
		ReplyToReseller:  replyToResellerPtr,
		PerintahProvider: perintahProviderPtr,
	}

	data, err := h.usecase.GetOutbox(r.Context(), tenant, dbSource, filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.Success(w, r, map[string]any{"items": data})
}

func (h *OutboxHandler) CreateOutbox(w http.ResponseWriter, r *http.Request) {
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

	tipePenerima := payload.TipePenerima
	if tipePenerima == "" {
		tipePenerima = "P"
	}
	status := int16(payload.Status)
	isPerintah := payload.IsPerintah
	bebasBiaya := int16(payload.BebasBiaya)

	data := domain.Outbox{
		Pesan:         payload.Pesan,
		Penerima:      payload.Penerima,
		TipePenerima:  tipePenerima,
		Status:        status,
		IsPerintah:    isPerintah,
		BebasBiaya:    bebasBiaya,
		KodeInbox:     payload.KodeInbox,
		KodeTransaksi: payload.KodeTransaksi,
		KodeReseller:  payload.KodeReseller,
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
	response.SuccessCreated(w, r, map[string]any{"message": "Sukses"})
}

func (h *OutboxHandler) UpdateOutbox(w http.ResponseWriter, r *http.Request) {
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
	response.Success(w, r, map[string]any{"message": "Sukses"})
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
