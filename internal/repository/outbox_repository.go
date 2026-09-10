package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/pkg/database"
)

type outboxRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *outboxRepository) GetOutboxPG(tenant string, filter domain.OutboxFilter) ([]domain.Outbox, int64, error) {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return nil, 0, fmt.Errorf("tenant tidak ditemukan")
	}

	query := `SELECT kode, tgl_entri, penerima, tipe_penerima, pesan, status, tgl_status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim FROM outbox WHERE 1=1`
	args := []any{}
	argID := 1

	if filter.StartDate != nil {
		query += fmt.Sprintf(` AND tgl_entri >= $%d`, argID)
		args = append(args, *filter.StartDate)
		argID++
	}
	if filter.EndDate != nil {
		query += fmt.Sprintf(` AND tgl_entri <= $%d`, argID)
		args = append(args, *filter.EndDate)
		argID++
	}
	if filter.Reseller != nil {
		query += fmt.Sprintf(` AND kode_reseller = $%d`, argID)
		args = append(args, *filter.Reseller)
		argID++
	}
	if filter.Penerima != nil {
		query += fmt.Sprintf(` AND penerima = $%d`, argID)
		args = append(args, *filter.Penerima)
		argID++
	}
	if filter.Tipe != nil {
		query += fmt.Sprintf(` AND tipe_penerima = $%d`, argID)
		args = append(args, *filter.Tipe)
		argID++
	}
	if filter.Status != nil {
		query += fmt.Sprintf(` AND status = $%d`, argID)
		args = append(args, *filter.Status)
		argID++
	}
	if filter.Pesan != "" {
		query += fmt.Sprintf(` AND pesan ILIKE $%d`, argID)
		args = append(args, "%"+filter.Pesan+"%")
		argID++
	}
	if filter.ReplyToReseller != nil && *filter.ReplyToReseller {
		query += ` AND kode_reseller IS NOT NULL`
	}
	if filter.PerintahProvider != nil && *filter.PerintahProvider {
		query += ` AND is_perintah = 1`
	}
	if filter.Search != "" {
		query += fmt.Sprintf(` AND (pesan ILIKE $%d OR penerima ILIKE $%d)`, argID, argID)
		args = append(args, "%"+filter.Search+"%")
		argID++
	}
	if filter.Cursor > 0 {
		query += fmt.Sprintf(` AND kode < $%d`, argID)
		args = append(args, filter.Cursor)
		argID++
	}

	query += fmt.Sprintf(` ORDER BY kode DESC LIMIT $%d`, argID)
	args = append(args, filter.Limit)

	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var outboxes []domain.Outbox
	var lastCursor int64

	for rows.Next() {
		var o domain.Outbox
		if err := rows.Scan(&o.Kode, &o.TglEntri, &o.Penerima, &o.TipePenerima, &o.Pesan, &o.Status, &o.TglStatus, &o.KodeInbox, &o.KodeTransaksi, &o.KodeReseller, &o.BebasBiaya, &o.IsPerintah, &o.KodeModul, &o.Prioritas, &o.ModulProses, &o.Pengirim, &o.KodeTerminal, &o.CtrKirim); err != nil {
			return nil, 0, err
		}
		outboxes = append(outboxes, o)
		lastCursor = o.Kode
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return outboxes, lastCursor, nil
}

func (r *outboxRepository) GetOutboxMS(tenant string, filter domain.OutboxFilter) ([]domain.Outbox, int64, error) {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return nil, 0, fmt.Errorf("tenant tidak ditemukan")
	}

	query := `SELECT TOP (@limit) kode, tgl_entri, penerima, tipe_penerima, pesan, status, tgl_status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim FROM outbox WHERE 1=1`
	namedArgs := []any{sql.Named("limit", filter.Limit)}

	if filter.StartDate != nil {
		query += ` AND tgl_entri >= @startDate`
		namedArgs = append(namedArgs, sql.Named("startDate", *filter.StartDate))
	}
	if filter.EndDate != nil {
		query += ` AND tgl_entri <= @endDate`
		namedArgs = append(namedArgs, sql.Named("endDate", *filter.EndDate))
	}
	if filter.Reseller != nil {
		query += ` AND kode_reseller = @reseller`
		namedArgs = append(namedArgs, sql.Named("reseller", *filter.Reseller))
	}
	if filter.Penerima != nil {
		query += ` AND penerima = @penerima`
		namedArgs = append(namedArgs, sql.Named("penerima", *filter.Penerima))
	}
	if filter.Tipe != nil {
		query += ` AND tipe_penerima = @tipe`
		namedArgs = append(namedArgs, sql.Named("tipe", *filter.Tipe))
	}
	if filter.Status != nil {
		query += ` AND status = @status`
		namedArgs = append(namedArgs, sql.Named("status", *filter.Status))
	}
	if filter.Pesan != "" {
		query += ` AND pesan LIKE '%' + @pesan + '%'`
		namedArgs = append(namedArgs, sql.Named("pesan", filter.Pesan))
	}
	if filter.ReplyToReseller != nil && *filter.ReplyToReseller {
		query += ` AND kode_reseller IS NOT NULL`
	}
	if filter.PerintahProvider != nil && *filter.PerintahProvider {
		query += ` AND is_perintah = 1`
	}
	if filter.Search != "" {
		query += ` AND (pesan LIKE '%' + @search + '%' OR penerima LIKE '%' + @search + '%')`
		namedArgs = append(namedArgs, sql.Named("search", filter.Search))
	}
	if filter.Cursor > 0 {
		query += ` AND kode < @cursor`
		namedArgs = append(namedArgs, sql.Named("cursor", filter.Cursor))
	}

	query += ` ORDER BY kode DESC`

	rows, err := db.Query(query, namedArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var outboxes []domain.Outbox
	var lastCursor int64

	for rows.Next() {
		var o domain.Outbox
		if err := rows.Scan(&o.Kode, &o.TglEntri, &o.Penerima, &o.TipePenerima, &o.Pesan, &o.Status, &o.TglStatus, &o.KodeInbox, &o.KodeTransaksi, &o.KodeReseller, &o.BebasBiaya, &o.IsPerintah, &o.KodeModul, &o.Prioritas, &o.ModulProses, &o.Pengirim, &o.KodeTerminal, &o.CtrKirim); err != nil {
			return nil, 0, err
		}
		outboxes = append(outboxes, o)
		lastCursor = o.Kode
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return outboxes, lastCursor, nil
}

func (r *outboxRepository) InsertPG(tenant string, data domain.Outbox) error {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	query := `INSERT INTO outbox (tgl_entri, penerima, tipe_penerima, pesan, status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`
	_, err := db.Exec(context.Background(), query, time.Now(), data.Penerima, data.TipePenerima, data.Pesan, data.Status, data.KodeInbox, data.KodeTransaksi, data.KodeReseller, data.BebasBiaya, data.IsPerintah, data.KodeModul, data.Prioritas, data.ModulProses, data.Pengirim, data.KodeTerminal, data.CtrKirim)
	return err
}

func (r *outboxRepository) InsertMS(tenant string, data domain.Outbox) error {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	query := `INSERT INTO outbox (tgl_entri, penerima, tipe_penerima, pesan, status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim) VALUES (@tgl_entri, @penerima, @tipe_penerima, @pesan, @status, @kode_inbox, @kode_transaksi, @kode_reseller, @bebas_biaya, @is_perintah, @kode_modul, @prioritas, @modul_proses, @pengirim, @kode_terminal, @ctr_kirim)`
	_, err := db.Exec(query, sql.Named("tgl_entri", time.Now()), sql.Named("penerima", data.Penerima), sql.Named("tipe_penerima", data.TipePenerima), sql.Named("pesan", data.Pesan), sql.Named("status", data.Status), sql.Named("kode_inbox", data.KodeInbox), sql.Named("kode_transaksi", data.KodeTransaksi), sql.Named("kode_reseller", data.KodeReseller), sql.Named("bebas_biaya", data.BebasBiaya), sql.Named("is_perintah", data.IsPerintah), sql.Named("kode_modul", data.KodeModul), sql.Named("prioritas", data.Prioritas), sql.Named("modul_proses", data.ModulProses), sql.Named("pengirim", data.Pengirim), sql.Named("kode_terminal", data.KodeTerminal), sql.Named("ctr_kirim", data.CtrKirim))
	return err
}

func (r *outboxRepository) UpdatePG(tenant string, kode int64, req dto.UpdateOutboxRequest) error {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	query := `UPDATE outbox SET
		tgl_status = $1,
		penerima = COALESCE($2, penerima),
		tipe_penerima = COALESCE($3, tipe_penerima),
		pesan = COALESCE($4, pesan),
		status = COALESCE($5, status),
		bebas_biaya = COALESCE($6, bebas_biaya),
		kode_inbox = COALESCE($7, kode_inbox),
		kode_transaksi = COALESCE($8, kode_transaksi),
		kode_reseller = COALESCE($9, kode_reseller),
		is_perintah = COALESCE($10, is_perintah),
		kode_modul = COALESCE($11, kode_modul),
		prioritas = COALESCE($12, prioritas),
		modul_proses = COALESCE($13, modul_proses),
		pengirim = COALESCE($14, pengirim),
		kode_terminal = COALESCE($15, kode_terminal),
		ctr_kirim = COALESCE($16, ctr_kirim)
	WHERE kode = $17`
	_, err := db.Exec(context.Background(), query, time.Now(), req.Penerima, req.TipePenerima, req.Pesan, req.Status, req.BebasBiaya, req.KodeInbox, req.KodeTransaksi, req.KodeReseller, req.IsPerintah, req.KodeModul, req.Prioritas, req.ModulProses, req.Pengirim, req.KodeTerminal, req.CtrKirim, kode)
	return err
}

func (r *outboxRepository) UpdateMS(tenant string, kode int64, req dto.UpdateOutboxRequest) error {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	query := `UPDATE outbox SET
		tgl_status = @tgl_status,
		penerima = COALESCE(@penerima, penerima),
		tipe_penerima = COALESCE(@tipe_penerima, tipe_penerima),
		pesan = COALESCE(@pesan, pesan),
		status = COALESCE(@status, status),
		bebas_biaya = COALESCE(@bebas_biaya, bebas_biaya),
		kode_inbox = COALESCE(@kode_inbox, kode_inbox),
		kode_transaksi = COALESCE(@kode_transaksi, kode_transaksi),
		kode_reseller = COALESCE(@kode_reseller, kode_reseller),
		is_perintah = COALESCE(@is_perintah, is_perintah),
		kode_modul = COALESCE(@kode_modul, kode_modul),
		prioritas = COALESCE(@prioritas, prioritas),
		modul_proses = COALESCE(@modul_proses, modul_proses),
		pengirim = COALESCE(@pengirim, pengirim),
		kode_terminal = COALESCE(@kode_terminal, kode_terminal),
		ctr_kirim = COALESCE(@ctr_kirim, ctr_kirim)
	WHERE kode = @kode`
	_, err := db.Exec(query,
		sql.Named("tgl_status", time.Now()),
		sql.Named("penerima", req.Penerima),
		sql.Named("tipe_penerima", req.TipePenerima),
		sql.Named("pesan", req.Pesan),
		sql.Named("status", req.Status),
		sql.Named("bebas_biaya", req.BebasBiaya),
		sql.Named("kode_inbox", req.KodeInbox),
		sql.Named("kode_transaksi", req.KodeTransaksi),
		sql.Named("kode_reseller", req.KodeReseller),
		sql.Named("is_perintah", req.IsPerintah),
		sql.Named("kode_modul", req.KodeModul),
		sql.Named("prioritas", req.Prioritas),
		sql.Named("modul_proses", req.ModulProses),
		sql.Named("pengirim", req.Pengirim),
		sql.Named("kode_terminal", req.KodeTerminal),
		sql.Named("ctr_kirim", req.CtrKirim),
		sql.Named("kode", kode),
	)
	return err
}

func NewOutboxRepository(dbRegistry *database.DBRegistry) domain.OutboxRepository {
	return &outboxRepository{dbRegistry: dbRegistry}
}
