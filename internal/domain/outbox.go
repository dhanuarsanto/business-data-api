package domain

import (
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
	KodeTransaksi *int32     `json:"kode_transaksi,omitempty"`
	KodeReseller  *string    `json:"kode_reseller,omitempty"`
	BebasBiaya    int16      `json:"bebas_biaya"`
	IsPerintah    *int16     `json:"is_perintah,omitempty"`
	KodeModul     *int32     `json:"kode_modul,omitempty"`
	Prioritas     *int16     `json:"prioritas,omitempty"`
	ModulProses   *string    `json:"modul_proses,omitempty"`
	Pengirim      *string    `json:"pengirim,omitempty"`
	KodeTerminal  *int32     `json:"kode_terminal,omitempty"`
	CtrKirim      *int16     `json:"ctr_kirim,omitempty"`
}

type OutboxFilter struct {
	Status *int16
	Search string
	Cursor int64
	Limit  int
}

type OutboxRepository interface {
	GetOutboxPG(tenant string, filter OutboxFilter) ([]Outbox, int64, error)
	GetOutboxMS(tenant string, filter OutboxFilter) ([]Outbox, int64, error)
	InsertPG(tenant string, data Outbox) error
	InsertMS(tenant string, data Outbox) error
	UpdatePG(tenant string, kode int64, req dto.UpdateOutboxRequest) error
	UpdateMS(tenant string, kode int64, req dto.UpdateOutboxRequest) error
}
