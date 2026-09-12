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

func (r *outboxRepository) GetOutboxPG(ctx context.Context, tenant string, filter domain.OutboxFilter) ([]domain.Outbox, int, bool, error) {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return nil, 0, false, fmt.Errorf("tenant tidak ditemukan")
	}

	whereClause := ` WHERE 1=1`
	args := []any{}
	argID := 1

	if filter.StartDate != nil {
		whereClause += fmt.Sprintf(` AND tgl_entri >= $%d`, argID)
		args = append(args, *filter.StartDate)
		argID++
	}
	if filter.EndDate != nil {
		whereClause += fmt.Sprintf(` AND tgl_entri <= $%d`, argID)
		args = append(args, *filter.EndDate)
		argID++
	}
	if filter.Reseller != nil {
		whereClause += fmt.Sprintf(` AND kode_reseller = $%d`, argID)
		args = append(args, *filter.Reseller)
		argID++
	}
	if filter.Penerima != nil {
		whereClause += fmt.Sprintf(` AND penerima = $%d`, argID)
		args = append(args, *filter.Penerima)
		argID++
	}
	if filter.Tipe != nil {
		whereClause += fmt.Sprintf(` AND tipe_penerima = $%d`, argID)
		args = append(args, *filter.Tipe)
		argID++
	}
	if filter.Status != nil {
		whereClause += fmt.Sprintf(` AND status = $%d`, argID)
		args = append(args, *filter.Status)
		argID++
	}
	if filter.Pesan != "" {
		whereClause += fmt.Sprintf(` AND pesan ILIKE $%d`, argID)
		args = append(args, "%"+filter.Pesan+"%")
		argID++
	}
	if filter.ReplyToReseller != nil && *filter.ReplyToReseller {
		whereClause += ` AND kode_reseller IS NOT NULL`
	}
	if filter.PerintahProvider != nil && *filter.PerintahProvider {
		whereClause += ` AND is_perintah = 1`
	}
	if filter.Search != "" {
		whereClause += fmt.Sprintf(` AND (pesan ILIKE $%d OR penerima ILIKE $%d)`, argID, argID)
		args = append(args, "%"+filter.Search+"%")
		argID++
	}

	var totalData int
	if whereClause == " WHERE 1=1" {
		err := db.QueryRow(ctx, `SELECT COALESCE(reltuples::bigint, 0) FROM pg_class WHERE oid = 'outbox'::regclass`).Scan(&totalData)
		if err != nil {
			return nil, 0, false, err
		}
	} else {
		err := db.QueryRow(ctx, `SELECT COUNT(*) FROM outbox`+whereClause, args...).Scan(&totalData)
		if err != nil {
			return nil, 0, false, err
		}
	}

	if filter.Cursor > 0 {
		whereClause += fmt.Sprintf(` AND kode < $%d`, argID)
		args = append(args, filter.Cursor)
		argID++
	}

	query := `SELECT kode, tgl_entri, penerima, tipe_penerima, pesan, status, tgl_status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim FROM outbox` + whereClause
	query += fmt.Sprintf(` ORDER BY kode DESC LIMIT $%d`, argID)
	args = append(args, filter.Limit+1)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	var outboxes []domain.Outbox

	for rows.Next() {
		var o domain.Outbox
		if err := rows.Scan(&o.Kode, &o.TglEntri, &o.Penerima, &o.TipePenerima, &o.Pesan, &o.Status, &o.TglStatus, &o.KodeInbox, &o.KodeTransaksi, &o.KodeReseller, &o.BebasBiaya, &o.IsPerintah, &o.KodeModul, &o.Prioritas, &o.ModulProses, &o.Pengirim, &o.KodeTerminal, &o.CtrKirim); err != nil {
			return nil, 0, false, err
		}
		outboxes = append(outboxes, o)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, false, err
	}

	hasNextPage := len(outboxes) > filter.Limit
	if hasNextPage {
		outboxes = outboxes[:filter.Limit]
	}

	return outboxes, totalData, hasNextPage, nil
}

func (r *outboxRepository) GetOutboxMS(ctx context.Context, tenant string, filter domain.OutboxFilter) ([]domain.Outbox, int, bool, error) {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return nil, 0, false, fmt.Errorf("tenant tidak ditemukan")
	}

	whereClause := ` WHERE 1=1`
	namedArgs := []any{}

	if filter.StartDate != nil {
		whereClause += ` AND tgl_entri >= @startDate`
		namedArgs = append(namedArgs, sql.Named("startDate", *filter.StartDate))
	}
	if filter.EndDate != nil {
		whereClause += ` AND tgl_entri <= @endDate`
		namedArgs = append(namedArgs, sql.Named("endDate", *filter.EndDate))
	}
	if filter.Reseller != nil {
		whereClause += ` AND kode_reseller = @reseller`
		namedArgs = append(namedArgs, sql.Named("reseller", *filter.Reseller))
	}
	if filter.Penerima != nil {
		whereClause += ` AND penerima = @penerima`
		namedArgs = append(namedArgs, sql.Named("penerima", *filter.Penerima))
	}
	if filter.Tipe != nil {
		whereClause += ` AND tipe_penerima = @tipe`
		namedArgs = append(namedArgs, sql.Named("tipe", *filter.Tipe))
	}
	if filter.Status != nil {
		whereClause += ` AND status = @status`
		namedArgs = append(namedArgs, sql.Named("status", *filter.Status))
	}
	if filter.Pesan != "" {
		whereClause += ` AND pesan LIKE '%' + @pesan + '%'`
		namedArgs = append(namedArgs, sql.Named("pesan", filter.Pesan))
	}
	if filter.ReplyToReseller != nil && *filter.ReplyToReseller {
		whereClause += ` AND kode_reseller IS NOT NULL`
	}
	if filter.PerintahProvider != nil && *filter.PerintahProvider {
		whereClause += ` AND is_perintah = 1`
	}
	if filter.Search != "" {
		whereClause += ` AND (pesan LIKE '%' + @search + '%' OR penerima LIKE '%' + @search + '%')`
		namedArgs = append(namedArgs, sql.Named("search", filter.Search))
	}

	var totalData int
	if whereClause == " WHERE 1=1" {
		err := db.QueryRowContext(ctx, `SELECT COALESCE(SUM(rows), 0) FROM sys.partitions WHERE object_id = OBJECT_ID('outbox') AND index_id IN (0, 1)`).Scan(&totalData)
		if err != nil {
			return nil, 0, false, err
		}
	} else {
		err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox`+whereClause, namedArgs...).Scan(&totalData)
		if err != nil {
			return nil, 0, false, err
		}
	}

	if filter.Cursor > 0 {
		whereClause += ` AND kode < @cursor`
		namedArgs = append(namedArgs, sql.Named("cursor", filter.Cursor))
	}

	namedArgs = append(namedArgs, sql.Named("limit", filter.Limit+1))
	query := `SELECT TOP (@limit) kode, tgl_entri, penerima, tipe_penerima, pesan, status, tgl_status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim FROM outbox` + whereClause + ` ORDER BY kode DESC`

	rows, err := db.QueryContext(ctx, query, namedArgs...)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	var outboxes []domain.Outbox

	for rows.Next() {
		var o domain.Outbox
		if err := rows.Scan(&o.Kode, &o.TglEntri, &o.Penerima, &o.TipePenerima, &o.Pesan, &o.Status, &o.TglStatus, &o.KodeInbox, &o.KodeTransaksi, &o.KodeReseller, &o.BebasBiaya, &o.IsPerintah, &o.KodeModul, &o.Prioritas, &o.ModulProses, &o.Pengirim, &o.KodeTerminal, &o.CtrKirim); err != nil {
			return nil, 0, false, err
		}
		outboxes = append(outboxes, o)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, false, err
	}

	hasNextPage := len(outboxes) > filter.Limit
	if hasNextPage {
		outboxes = outboxes[:filter.Limit]
	}

	return outboxes, totalData, hasNextPage, nil
}

func (r *outboxRepository) InsertPG(ctx context.Context, tenant string, data domain.Outbox) error {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	query := `INSERT INTO outbox (tgl_entri, penerima, tipe_penerima, pesan, status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`
	_, err := db.Exec(ctx, query, time.Now(), data.Penerima, data.TipePenerima, data.Pesan, data.Status, data.KodeInbox, data.KodeTransaksi, data.KodeReseller, data.BebasBiaya, data.IsPerintah, data.KodeModul, data.Prioritas, data.ModulProses, data.Pengirim, data.KodeTerminal, data.CtrKirim)
	return err
}

func (r *outboxRepository) InsertMS(ctx context.Context, tenant string, data domain.Outbox) error {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	query := `INSERT INTO outbox (tgl_entri, penerima, tipe_penerima, pesan, status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim) VALUES (@tgl_entri, @penerima, @tipe_penerima, @pesan, @status, @kode_inbox, @kode_transaksi, @kode_reseller, @bebas_biaya, @is_perintah, @kode_modul, @prioritas, @modul_proses, @pengirim, @kode_terminal, @ctr_kirim)`
	_, err := db.ExecContext(ctx, query, sql.Named("tgl_entri", time.Now()), sql.Named("penerima", data.Penerima), sql.Named("tipe_penerima", data.TipePenerima), sql.Named("pesan", data.Pesan), sql.Named("status", data.Status), sql.Named("kode_inbox", data.KodeInbox), sql.Named("kode_transaksi", data.KodeTransaksi), sql.Named("kode_reseller", data.KodeReseller), sql.Named("bebas_biaya", data.BebasBiaya), sql.Named("is_perintah", data.IsPerintah), sql.Named("kode_modul", data.KodeModul), sql.Named("prioritas", data.Prioritas), sql.Named("modul_proses", data.ModulProses), sql.Named("pengirim", data.Pengirim), sql.Named("kode_terminal", data.KodeTerminal), sql.Named("ctr_kirim", data.CtrKirim))
	return err
}

func (r *outboxRepository) UpdatePG(ctx context.Context, tenant string, kode int64, req dto.UpdateOutboxRequest) error {
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
	_, err := db.Exec(ctx, query, time.Now(), req.Penerima, req.TipePenerima, req.Pesan, req.Status, req.BebasBiaya, req.KodeInbox, req.KodeTransaksi, req.KodeReseller, req.IsPerintah, req.KodeModul, req.Prioritas, req.ModulProses, req.Pengirim, req.KodeTerminal, req.CtrKirim, kode)
	return err
}

func (r *outboxRepository) UpdateMS(ctx context.Context, tenant string, kode int64, req dto.UpdateOutboxRequest) error {
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
	_, err := db.ExecContext(ctx, query,
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
