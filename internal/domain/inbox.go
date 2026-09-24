package domain

import (
	"context"
	"time"

	"go.internal/business-data-api/internal/dto"
)

type Inbox struct {
	Kode          int64      `json:"kode"`
	TglEntri      time.Time  `json:"tgl_entri"`
	TglStatus     *time.Time `json:"tgl_status,omitempty"`
	Pengirim      string     `json:"pengirim"`
	TipePengirim  string     `json:"tipe_pengirim"`
	Penerima      *string    `json:"penerima,omitempty"`
	Pesan         string     `json:"pesan"`
	Status        int16      `json:"status"`
	KodeTerminal  *int       `json:"kode_terminal,omitempty"`
	KodeReseller  *string    `json:"kode_reseller,omitempty"`
	KodeTransaksi *int       `json:"kode_transaksi,omitempty"`
	IsJawaban     int16      `json:"is_jawaban"`
	ServiceCenter *string    `json:"service_center,omitempty"`
	IsCs          *int16     `json:"is_cs,omitempty"`
	KodeJawabanCs *int64     `json:"kode_jawaban_cs,omitempty"`
	Hash          *string    `json:"hash,omitempty"`
}

type InboxFilter struct {
	StartDate           *time.Time
	EndDate             *time.Time
	PageSize            int
	LimitTotal          *int
	LowerBound          int64
	Terminal            *int
	Reseller            *string
	Pengirim            *string
	Tipe                *string
	Status              *int16
	Pesan               string
	RequestFromReseller *bool
	JawabanFromProvider *bool
	Cursor              int64
}

type InboxRepository interface {
	Get(ctx context.Context, tenant string, filter InboxFilter) ([]Inbox, bool, error)
	LowerBound(ctx context.Context, tenant string, filter InboxFilter) (int64, error)
	Insert(ctx context.Context, tenant string, data Inbox) error
	Update(ctx context.Context, tenant string, kode int64, req dto.UpdateInboxRequest) error
}
