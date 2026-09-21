package dto

import "time"

type ResellerDropdown struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

type CreateResellerRequest struct {
	Kode              string     `json:"kode"`
	Nama              string     `json:"nama"`
	Saldo             float64    `json:"saldo"`
	Alamat            *string    `json:"alamat,omitempty"`
	Pin               *string    `json:"pin,omitempty"`
	Aktif             int16      `json:"aktif"`
	KodeUpline        *string    `json:"kode_upline,omitempty"`
	KodeLevel         string     `json:"kode_level"`
	Keterangan        *string    `json:"keterangan,omitempty"`
	TglDaftar         *time.Time `json:"tgl_daftar,omitempty"`
	SaldoMinimal      float64    `json:"saldo_minimal"`
	TglAktivitas      *time.Time `json:"tgl_aktivitas,omitempty"`
	PengingatSaldo    *float64   `json:"pengingat_saldo,omitempty"`
	FPengingatSaldo   int16      `json:"f_pengingat_saldo"`
	NamaPemilik       *string    `json:"nama_pemilik,omitempty"`
	KodeArea          *string    `json:"kode_area,omitempty"`
	TglPengingatSaldo *time.Time `json:"tgl_pengingat_saldo,omitempty"`
	Markup            *float64   `json:"markup,omitempty"`
	Oid               *string    `json:"oid,omitempty"`
	Poin              *int       `json:"poin,omitempty"`
	AlamatIP          *string    `json:"alamat_ip,omitempty"`
	PasswordIP        *string    `json:"password_ip,omitempty"`
	URLReport         *string    `json:"url_report,omitempty"`
	TglData           *time.Time `json:"tgl_data,omitempty"`
	Suspend           *int16     `json:"suspend,omitempty"`
	IPNoSign          *int16     `json:"ip_no_sign,omitempty"`
	Deleted           *int16     `json:"deleted,omitempty"`
	NomorKTP          *string    `json:"nomor_ktp,omitempty"`
	NPWP              *string    `json:"npwp,omitempty"`
	HarusUbahPin      *int16     `json:"harus_ubah_pin,omitempty"`
	NomorHP           *string    `json:"nomor_hp,omitempty"`
	Email             *string    `json:"email,omitempty"`
	BlokProduk        *string    `json:"blok_produk,omitempty"`
	KodeReferral      *string    `json:"kode_referral,omitempty"`
	Komisi            *float64   `json:"komisi,omitempty"`
	KodeDeposit       *int       `json:"kode_deposit,omitempty"`
	BeritaTransfer    *string    `json:"berita_transfer,omitempty"`
	BeritaBank        *string    `json:"berita_bank,omitempty"`
}

type UpdateResellerRequest struct {
	Nama              *string    `json:"nama,omitempty"`
	Saldo             *float64   `json:"saldo,omitempty"`
	Alamat            *string    `json:"alamat,omitempty"`
	Pin               *string    `json:"pin,omitempty"`
	Aktif             *int16     `json:"aktif,omitempty"`
	KodeUpline        *string    `json:"kode_upline,omitempty"`
	KodeLevel         *string    `json:"kode_level,omitempty"`
	Keterangan        *string    `json:"keterangan,omitempty"`
	TglDaftar         *time.Time `json:"tgl_daftar,omitempty"`
	SaldoMinimal      *float64   `json:"saldo_minimal,omitempty"`
	TglAktivitas      *time.Time `json:"tgl_aktivitas,omitempty"`
	PengingatSaldo    *float64   `json:"pengingat_saldo,omitempty"`
	FPengingatSaldo   *int16     `json:"f_pengingat_saldo,omitempty"`
	NamaPemilik       *string    `json:"nama_pemilik,omitempty"`
	KodeArea          *string    `json:"kode_area,omitempty"`
	TglPengingatSaldo *time.Time `json:"tgl_pengingat_saldo,omitempty"`
	Markup            *float64   `json:"markup,omitempty"`
	Oid               *string    `json:"oid,omitempty"`
	Poin              *int       `json:"poin,omitempty"`
	AlamatIP          *string    `json:"alamat_ip,omitempty"`
	PasswordIP        *string    `json:"password_ip,omitempty"`
	URLReport         *string    `json:"url_report,omitempty"`
	TglData           *time.Time `json:"tgl_data,omitempty"`
	Suspend           *int16     `json:"suspend,omitempty"`
	IPNoSign          *int16     `json:"ip_no_sign,omitempty"`
	Deleted           *int16     `json:"deleted,omitempty"`
	NomorKTP          *string    `json:"nomor_ktp,omitempty"`
	NPWP              *string    `json:"npwp,omitempty"`
	HarusUbahPin      *int16     `json:"harus_ubah_pin,omitempty"`
	NomorHP           *string    `json:"nomor_hp,omitempty"`
	Email             *string    `json:"email,omitempty"`
	BlokProduk        *string    `json:"blok_produk,omitempty"`
	KodeReferral      *string    `json:"kode_referral,omitempty"`
	Komisi            *float64   `json:"komisi,omitempty"`
	KodeDeposit       *int       `json:"kode_deposit,omitempty"`
	BeritaTransfer    *string    `json:"berita_transfer,omitempty"`
	BeritaBank        *string    `json:"berita_bank,omitempty"`
}