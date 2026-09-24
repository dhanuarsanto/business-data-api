package dto

type CreateOutboxRequest struct {
	Penerima      string  `json:"penerima"`
	TipePenerima  string  `json:"tipe_penerima"`
	Pesan         string  `json:"pesan"`
	Status        int16   `json:"status"`
	BebasBiaya    int16   `json:"bebas_biaya"`
	KodeInbox     *int64  `json:"kode_inbox,omitempty"`
	KodeTransaksi *int    `json:"kode_transaksi,omitempty"`
	KodeReseller  *string `json:"kode_reseller,omitempty"`
	IsPerintah    *int16  `json:"is_perintah,omitempty"`
	KodeModul     *int    `json:"kode_modul,omitempty"`
	Prioritas     *int16  `json:"prioritas,omitempty"`
	ModulProses   *string `json:"modul_proses,omitempty"`
	Pengirim      *string `json:"pengirim,omitempty"`
	KodeTerminal  *int    `json:"kode_terminal,omitempty"`
	CtrKirim      *int16  `json:"ctr_kirim,omitempty"`
}

type UpdateOutboxRequest struct {
	Penerima      *string `json:"penerima,omitempty"`
	TipePenerima  *string `json:"tipe_penerima,omitempty"`
	Pesan         *string `json:"pesan,omitempty"`
	Status        *int16  `json:"status,omitempty"`
	BebasBiaya    *int16  `json:"bebas_biaya,omitempty"`
	KodeInbox     *int64  `json:"kode_inbox,omitempty"`
	KodeTransaksi *int    `json:"kode_transaksi,omitempty"`
	KodeReseller  *string `json:"kode_reseller,omitempty"`
	IsPerintah    *int16  `json:"is_perintah,omitempty"`
	KodeModul     *int    `json:"kode_modul,omitempty"`
	Prioritas     *int16  `json:"prioritas,omitempty"`
	ModulProses   *string `json:"modul_proses,omitempty"`
	Pengirim      *string `json:"pengirim,omitempty"`
	KodeTerminal  *int    `json:"kode_terminal,omitempty"`
	CtrKirim      *int16  `json:"ctr_kirim,omitempty"`
}
