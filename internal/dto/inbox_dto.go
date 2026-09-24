package dto

type CreateInboxRequest struct {
	Pesan         string  `json:"pesan"`
	Pengirim      string  `json:"pengirim"`
	Penerima      *string `json:"penerima,omitempty"`
	TipePengirim  string  `json:"tipe_pengirim,omitempty"`
	KodeTerminal  *int    `json:"kode_terminal,omitempty"`
	KodeReseller  *string `json:"kode_reseller,omitempty"`
	KodeTransaksi *int    `json:"kode_transaksi,omitempty"`
	ServiceCenter *string `json:"service_center,omitempty"`
	Status        *int16  `json:"status,omitempty"`
	IsJawaban     *int16  `json:"is_jawaban,omitempty"`
	IsCs          *int16  `json:"is_cs,omitempty"`
	KodeJawabanCs *int64  `json:"kode_jawaban_cs,omitempty"`
	Hash          *string `json:"hash,omitempty"`
}

type UpdateInboxRequest struct {
	Pesan         *string `json:"pesan,omitempty"`
	Status        *int16  `json:"status,omitempty"`
	Pengirim      *string `json:"pengirim,omitempty"`
	TipePengirim  *string `json:"tipe_pengirim,omitempty"`
	Penerima      *string `json:"penerima,omitempty"`
	KodeTerminal  *int    `json:"kode_terminal,omitempty"`
	KodeReseller  *string `json:"kode_reseller,omitempty"`
	KodeTransaksi *int    `json:"kode_transaksi,omitempty"`
	IsJawaban     *int16  `json:"is_jawaban,omitempty"`
	ServiceCenter *string `json:"service_center,omitempty"`
	IsCs          *int16  `json:"is_cs,omitempty"`
	KodeJawabanCs *int64  `json:"kode_jawaban_cs,omitempty"`
	Hash          *string `json:"hash,omitempty"`
}
