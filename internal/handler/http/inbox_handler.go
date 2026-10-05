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

type InboxHandler struct {
	usecase *usecase.InboxUsecase
}

func (h *InboxHandler) GetInbox(w http.ResponseWriter, r *http.Request) {
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

	terminalPtr, err := parseTerminalFilter(queryParams.Get("terminal"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	resellerPtr := parseStringFilter(queryParams.Get("reseller"))
	pengirimPtr := parseStringFilter(queryParams.Get("pengirim"))
	tipePtr := parseStringFilter(queryParams.Get("tipe"))

	statusPtr, statusMinPtr, err := parseStatusFilter(queryParams.Get("status"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	reqFromResellerPtr, err := parseBoolFilter(queryParams.Get("requestFromReseller"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	jawFromProviderPtr, err := parseBoolFilter(queryParams.Get("jawabanFromProvider"))
	if err != nil {
		response.Error(w, r, http.StatusBadRequest, err.Error())
		return
	}

	pesan := strings.TrimSpace(queryParams.Get("pesan"))
	if len(pesan) > maxStringLen {
		pesan = ""
	}

	filter := domain.InboxFilter{
		StartDate:           params.StartDate,
		EndDate:             params.EndDate,
		Limit:               params.Limit,
		Terminal:            terminalPtr,
		Reseller:            resellerPtr,
		Pengirim:            pengirimPtr,
		Tipe:                tipePtr,
		Status:              statusPtr,
		StatusMin:           statusMinPtr,
		Pesan:               pesan,
		RequestFromReseller: reqFromResellerPtr,
		JawabanFromProvider: jawFromProviderPtr,
	}

	data, err := h.usecase.GetInbox(r.Context(), tenant, dbSource, filter)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response.Success(w, r, map[string]any{"items": data})
}

func (h *InboxHandler) CreateInbox(w http.ResponseWriter, r *http.Request) {
	tenant := chi.URLParam(r, "tenant")
	dbSource := r.Header.Get("X-DB-Source")
	if dbSource == "" {
		dbSource = domain.SourcePostgres
	}

	var payload dto.CreateInboxRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	tipePengirim := payload.TipePengirim
	if tipePengirim == "" {
		tipePengirim = "S"
	}
	status := int16(0)
	if payload.Status != nil {
		status = *payload.Status
	}
	isJawaban := int16(0)
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

	if err := h.usecase.CreateInbox(r.Context(), tenant, dbSource, data); err != nil {
		writeError(w, r, err)
		return
	}
	response.SuccessCreated(w, r, map[string]any{"message": "Sukses"})
}

func (h *InboxHandler) UpdateInbox(w http.ResponseWriter, r *http.Request) {
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

	var payload dto.UpdateInboxRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		response.Error(w, r, http.StatusBadRequest, "Format JSON tidak valid")
		return
	}

	if err := h.usecase.UpdateInbox(r.Context(), tenant, dbSource, kode, payload); err != nil {
		writeError(w, r, err)
		return
	}
	response.Success(w, r, map[string]any{"message": "Sukses"})
}

func NewInboxHandler(usecase *usecase.InboxUsecase) *InboxHandler {
	return &InboxHandler{usecase: usecase}
}

func (h *InboxHandler) RegisterRoutes(_public, protected chi.Router, cfg *config.Config, roleMatrix map[string][]string) {
	protected.Route("/api/v1/{tenant}/inbox", func(inbox chi.Router) {
		inbox.Group(func(read chi.Router) {
			read.Use(api_middleware.RequireRole(roleMatrix["ReadInbox"]...))
			read.Get("/", h.GetInbox)
		})

		inbox.Group(func(write chi.Router) {
			write.Use(api_middleware.RequireRole(roleMatrix["WriteInbox"]...))
			write.Use(api_middleware.PostgresWriteGuard(cfg.PostgresWriteEnabled))
			write.Post("/", h.CreateInbox)
			write.Put("/{kode}", h.UpdateInbox)
		})
	})
}
