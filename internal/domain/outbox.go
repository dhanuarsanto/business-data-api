package domain

import (
	"context"
	"time"

	"go.internal/business-data-api/internal/dto"
)

type Outbox struct {
	Kode          int64      `json:"kode"`
	TglEntri      time.Time  `json:"tgl_entri"`
	Penerima      string     `json:"penerima"`
	TipePenerima  string     `json:"tipe_penerima"`
	Pesan         string     `json:"pesan"`
	Status        int16      `json:"status"`
	TglStatus     *time.Time `json:"tgl_status,omitempty"`
	KodeInbox     *int64     `json:"kode_inbox,omitempty"`
	KodeTransaksi *int       `json:"kode_transaksi,omitempty"`
	KodeReseller  *string    `json:"kode_reseller,omitempty"`
	BebasBiaya    int16      `json:"bebas_biaya"`
	IsPerintah    *int16     `json:"is_perintah,omitempty"`
	KodeModul     *int       `json:"kode_modul,omitempty"`
	Prioritas     *int16     `json:"prioritas,omitempty"`
	ModulProses   *string    `json:"modul_proses,omitempty"`
	Pengirim      *string    `json:"pengirim,omitempty"`
	KodeTerminal  *int       `json:"kode_terminal,omitempty"`
	CtrKirim      *int16     `json:"ctr_kirim,omitempty"`
}

type OutboxFilter struct {
	StartDate        *time.Time
	EndDate          *time.Time
	PageSize         int
	LimitTotal       *int
	LowerBound       int64
	Reseller         *string
	Penerima         *string
	Tipe             *string
	Status           *int16
	Pesan            string
	ReplyToReseller  *bool
	PerintahProvider *bool
	Cursor           int64
}

type OutboxRepository interface {
	Get(ctx context.Context, tenant string, filter OutboxFilter) ([]Outbox, bool, error)
	LowerBound(ctx context.Context, tenant string, filter OutboxFilter) (int64, error)
	Insert(ctx context.Context, tenant string, data Outbox) error
	Update(ctx context.Context, tenant string, kode int64, req dto.UpdateOutboxRequest) error
}
