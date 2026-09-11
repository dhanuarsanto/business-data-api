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

type inboxRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *inboxRepository) GetInboxPG(tenant string, filter domain.InboxFilter) ([]domain.Inbox, int, bool, error) {
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
	if filter.Terminal != nil {
		whereClause += fmt.Sprintf(` AND kode_terminal = $%d`, argID)
		args = append(args, *filter.Terminal)
		argID++
	}
	if filter.Reseller != nil {
		whereClause += fmt.Sprintf(` AND kode_reseller = $%d`, argID)
		args = append(args, *filter.Reseller)
		argID++
	}
	if filter.Pengirim != nil {
		whereClause += fmt.Sprintf(` AND pengirim = $%d`, argID)
		args = append(args, *filter.Pengirim)
		argID++
	}
	if filter.Tipe != nil {
		whereClause += fmt.Sprintf(` AND tipe_pengirim = $%d`, argID)
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
	if filter.RequestFromReseller != nil && *filter.RequestFromReseller {
		whereClause += ` AND kode_reseller IS NOT NULL`
	}
	if filter.JawabanFromProvider != nil && *filter.JawabanFromProvider {
		whereClause += ` AND is_jawaban = 1`
	}
	if filter.Search != "" {
		whereClause += fmt.Sprintf(` AND (pesan ILIKE $%d OR pengirim ILIKE $%d)`, argID, argID)
		args = append(args, "%"+filter.Search+"%")
		argID++
	}

	var totalData int
	if whereClause == " WHERE 1=1" {
		err := db.QueryRow(context.Background(), `SELECT COALESCE(reltuples::bigint, 0) FROM pg_class WHERE oid = 'inbox'::regclass`).Scan(&totalData)
		if err != nil {
			return nil, 0, false, err
		}
	} else {
		err := db.QueryRow(context.Background(), `SELECT COUNT(*) FROM inbox`+whereClause, args...).Scan(&totalData)
		if err != nil {
			return nil, 0, false, err
		}
	}

	if filter.Cursor > 0 {
		whereClause += fmt.Sprintf(` AND kode < $%d`, argID)
		args = append(args, filter.Cursor)
		argID++
	}

	query := `SELECT kode, tgl_entri, tgl_status, pengirim, tipe_pengirim, penerima, pesan, status, kode_terminal, kode_reseller, kode_transaksi, is_jawaban, service_center, is_cs, kode_jawaban_cs, hash FROM inbox` + whereClause
	query += fmt.Sprintf(` ORDER BY kode DESC LIMIT $%d`, argID)
	args = append(args, filter.Limit+1)

	rows, err := db.Query(context.Background(), query, args...)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	var inboxes []domain.Inbox

	for rows.Next() {
		var i domain.Inbox
		if err := rows.Scan(&i.Kode, &i.TglEntri, &i.TglStatus, &i.Pengirim, &i.TipePengirim, &i.Penerima, &i.Pesan, &i.Status, &i.KodeTerminal, &i.KodeReseller, &i.KodeTransaksi, &i.IsJawaban, &i.ServiceCenter, &i.IsCs, &i.KodeJawabanCs, &i.Hash); err != nil {
			return nil, 0, false, err
		}
		inboxes = append(inboxes, i)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, false, err
	}

	hasNextPage := len(inboxes) > filter.Limit
	if hasNextPage {
		inboxes = inboxes[:filter.Limit]
	}

	return inboxes, totalData, hasNextPage, nil
}

func (r *inboxRepository) GetInboxMS(tenant string, filter domain.InboxFilter) ([]domain.Inbox, int, bool, error) {
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
	if filter.Terminal != nil {
		whereClause += ` AND kode_terminal = @terminal`
		namedArgs = append(namedArgs, sql.Named("terminal", *filter.Terminal))
	}
	if filter.Reseller != nil {
		whereClause += ` AND kode_reseller = @reseller`
		namedArgs = append(namedArgs, sql.Named("reseller", *filter.Reseller))
	}
	if filter.Pengirim != nil {
		whereClause += ` AND pengirim = @pengirim`
		namedArgs = append(namedArgs, sql.Named("pengirim", *filter.Pengirim))
	}
	if filter.Tipe != nil {
		whereClause += ` AND tipe_pengirim = @tipe`
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
	if filter.RequestFromReseller != nil && *filter.RequestFromReseller {
		whereClause += ` AND kode_reseller IS NOT NULL`
	}
	if filter.JawabanFromProvider != nil && *filter.JawabanFromProvider {
		whereClause += ` AND is_jawaban = 1`
	}
	if filter.Search != "" {
		whereClause += ` AND (pesan LIKE '%' + @search + '%' OR pengirim LIKE '%' + @search + '%')`
		namedArgs = append(namedArgs, sql.Named("search", filter.Search))
	}

	var totalData int
	if whereClause == " WHERE 1=1" {
		err := db.QueryRowContext(context.Background(), `SELECT COALESCE(SUM(rows), 0) FROM sys.partitions WHERE object_id = OBJECT_ID('inbox') AND index_id IN (0, 1)`).Scan(&totalData)
		if err != nil {
			return nil, 0, false, err
		}
	} else {
		err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM inbox`+whereClause, namedArgs...).Scan(&totalData)
		if err != nil {
			return nil, 0, false, err
		}
	}

	if filter.Cursor > 0 {
		whereClause += ` AND kode < @cursor`
		namedArgs = append(namedArgs, sql.Named("cursor", filter.Cursor))
	}

	namedArgs = append(namedArgs, sql.Named("limit", filter.Limit+1))
	query := `SELECT TOP (@limit) kode, tgl_entri, tgl_status, pengirim, tipe_pengirim, penerima, pesan, status, kode_terminal, kode_reseller, kode_transaksi, is_jawaban, service_center, is_cs, kode_jawaban_cs, hash FROM inbox` + whereClause + ` ORDER BY kode DESC`

	rows, err := db.Query(query, namedArgs...)
	if err != nil {
		return nil, 0, false, err
	}
	defer rows.Close()

	var inboxes []domain.Inbox

	for rows.Next() {
		var i domain.Inbox
		if err := rows.Scan(&i.Kode, &i.TglEntri, &i.TglStatus, &i.Pengirim, &i.TipePengirim, &i.Penerima, &i.Pesan, &i.Status, &i.KodeTerminal, &i.KodeReseller, &i.KodeTransaksi, &i.IsJawaban, &i.ServiceCenter, &i.IsCs, &i.KodeJawabanCs, &i.Hash); err != nil {
			return nil, 0, false, err
		}
		inboxes = append(inboxes, i)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, false, err
	}

	hasNextPage := len(inboxes) > filter.Limit
	if hasNextPage {
		inboxes = inboxes[:filter.Limit]
	}

	return inboxes, totalData, hasNextPage, nil
}

func (r *inboxRepository) InsertPG(tenant string, data domain.Inbox) error {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	_, err := db.Exec(context.Background(), `INSERT INTO inbox (tgl_entri, pengirim, tipe_pengirim, penerima, pesan, status, kode_terminal, kode_reseller, kode_transaksi, is_jawaban, service_center, is_cs, kode_jawaban_cs, hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`, time.Now(), data.Pengirim, data.TipePengirim, data.Penerima, data.Pesan, data.Status, data.KodeTerminal, data.KodeReseller, data.KodeTransaksi, data.IsJawaban, data.ServiceCenter, data.IsCs, data.KodeJawabanCs, data.Hash)
	return err
}

func (r *inboxRepository) InsertMS(tenant string, data domain.Inbox) error {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	_, err := db.Exec(`INSERT INTO inbox (tgl_entri, pengirim, tipe_pengirim, penerima, pesan, status, kode_terminal, kode_reseller, kode_transaksi, is_jawaban, service_center, is_cs, kode_jawaban_cs, hash) VALUES (@tgl_entri, @pengirim, @tipe_pengirim, @penerima, @pesan, @status, @kode_terminal, @kode_reseller, @kode_transaksi, @is_jawaban, @service_center, @is_cs, @kode_jawaban_cs, @hash)`, sql.Named("tgl_entri", time.Now()), sql.Named("pengirim", data.Pengirim), sql.Named("tipe_pengirim", data.TipePengirim), sql.Named("penerima", data.Penerima), sql.Named("pesan", data.Pesan), sql.Named("status", data.Status), sql.Named("kode_terminal", data.KodeTerminal), sql.Named("kode_reseller", data.KodeReseller), sql.Named("kode_transaksi", data.KodeTransaksi), sql.Named("is_jawaban", data.IsJawaban), sql.Named("service_center", data.ServiceCenter), sql.Named("is_cs", data.IsCs), sql.Named("kode_jawaban_cs", data.KodeJawabanCs), sql.Named("hash", data.Hash))
	return err
}

func (r *inboxRepository) UpdatePG(tenant string, kode int64, req dto.UpdateInboxRequest) error {
	db := r.dbRegistry.Postgres(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	query := `UPDATE inbox SET
		tgl_status = $1,
		pesan = COALESCE($2, pesan),
		status = COALESCE($3, status),
		pengirim = COALESCE($4, pengirim),
		tipe_pengirim = COALESCE($5, tipe_pengirim),
		penerima = COALESCE($6, penerima),
		kode_terminal = COALESCE($7, kode_terminal),
		kode_reseller = COALESCE($8, kode_reseller),
		kode_transaksi = COALESCE($9, kode_transaksi),
		is_jawaban = COALESCE($10, is_jawaban),
		service_center = COALESCE($11, service_center),
		is_cs = COALESCE($12, is_cs),
		kode_jawaban_cs = COALESCE($13, kode_jawaban_cs),
		hash = COALESCE($14, hash)
	WHERE kode = $15`
	_, err := db.Exec(context.Background(), query, time.Now(), req.Pesan, req.Status, req.Pengirim, req.TipePengirim, req.Penerima, req.KodeTerminal, req.KodeReseller, req.KodeTransaksi, req.IsJawaban, req.ServiceCenter, req.IsCs, req.KodeJawabanCs, req.Hash, kode)
	return err
}

func (r *inboxRepository) UpdateMS(tenant string, kode int64, req dto.UpdateInboxRequest) error {
	db := r.dbRegistry.MSSQL(tenant)
	if db == nil {
		return fmt.Errorf("tenant tidak ditemukan")
	}
	query := `UPDATE inbox SET
		tgl_status = @tgl_status,
		pesan = COALESCE(@pesan, pesan),
		status = COALESCE(@status, status),
		pengirim = COALESCE(@pengirim, pengirim),
		tipe_pengirim = COALESCE(@tipe_pengirim, tipe_pengirim),
		penerima = COALESCE(@penerima, penerima),
		kode_terminal = COALESCE(@kode_terminal, kode_terminal),
		kode_reseller = COALESCE(@kode_reseller, kode_reseller),
		kode_transaksi = COALESCE(@kode_transaksi, kode_transaksi),
		is_jawaban = COALESCE(@is_jawaban, is_jawaban),
		service_center = COALESCE(@service_center, service_center),
		is_cs = COALESCE(@is_cs, is_cs),
		kode_jawaban_cs = COALESCE(@kode_jawaban_cs, kode_jawaban_cs),
		hash = COALESCE(@hash, hash)
	WHERE kode = @kode`
	_, err := db.Exec(query,
		sql.Named("tgl_status", time.Now()),
		sql.Named("pesan", req.Pesan),
		sql.Named("status", req.Status),
		sql.Named("pengirim", req.Pengirim),
		sql.Named("tipe_pengirim", req.TipePengirim),
		sql.Named("penerima", req.Penerima),
		sql.Named("kode_terminal", req.KodeTerminal),
		sql.Named("kode_reseller", req.KodeReseller),
		sql.Named("kode_transaksi", req.KodeTransaksi),
		sql.Named("is_jawaban", req.IsJawaban),
		sql.Named("service_center", req.ServiceCenter),
		sql.Named("is_cs", req.IsCs),
		sql.Named("kode_jawaban_cs", req.KodeJawabanCs),
		sql.Named("hash", req.Hash),
		sql.Named("kode", kode),
	)
	return err
}

func NewInboxRepository(dbRegistry *database.DBRegistry) domain.InboxRepository {
	return &inboxRepository{dbRegistry: dbRegistry}
}
