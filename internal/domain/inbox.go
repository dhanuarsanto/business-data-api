package domain

import (
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
	Status        int        `json:"status"`
	KodeTerminal  *int       `json:"kode_terminal,omitempty"`
	KodeReseller  *string    `json:"kode_reseller,omitempty"`
	KodeTransaksi *int       `json:"kode_transaksi,omitempty"`
	IsJawaban     int        `json:"is_jawaban"`
	ServiceCenter *string    `json:"service_center,omitempty"`
	IsCs          *int       `json:"is_cs,omitempty"`
	KodeJawabanCs *int64     `json:"kode_jawaban_cs,omitempty"`
	Hash          *string    `json:"hash,omitempty"`
}

type InboxFilter struct {
	Status *int
	Search string
	Cursor int64
	Limit  int
}

type InboxRepository interface {
	GetInboxPG(tenant string, filter InboxFilter) ([]Inbox, int64, error)
	GetInboxMS(tenant string, filter InboxFilter) ([]Inbox, int64, error)
	InsertPG(tenant string, data Inbox) error
	InsertMS(tenant string, data Inbox) error
	UpdatePG(tenant string, kode int64, req dto.UpdateInboxRequest) error
	UpdateMS(tenant string, kode int64, req dto.UpdateInboxRequest) error
}
