package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/pkg/database"
)

type OutboxRepositories struct {
	PG domain.OutboxRepository
	MS domain.OutboxRepository
}

type outboxPGRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *outboxPGRepository) Get(ctx context.Context, tenant string, filter domain.OutboxFilter) ([]domain.Outbox, bool, error) {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return nil, false, err
	}

	f := filter
	f.StartDate = nil
	whereClause, args, argID := buildOutboxFilterPG(f)

	if filter.EndDate != nil {
		cut, err := bisectCutPG(ctx, db, "outbox", *filter.EndDate)
		if err != nil {
			return nil, false, err
		}
		whereClause = " WHERE 1=1 AND kode <= $" + strconv.Itoa(argID) + whereClause[len(" WHERE 1=1"):]
		args = append(args, cut+bisectSlack)
		argID++
	}

	if filter.StartDate != nil {
		cut, err := bisectCutStartPG(ctx, db, "outbox", *filter.StartDate)
		if err != nil {
			return nil, false, err
		}
		whereClause += fmt.Sprintf(` AND kode > $%d`, argID)
		args = append(args, cut)
		argID++
	}

	if filter.LowerBound > 0 {
		whereClause += fmt.Sprintf(` AND kode >= $%d`, argID)
		args = append(args, filter.LowerBound)
		argID++
	}

	if filter.Cursor > 0 {
		whereClause += fmt.Sprintf(` AND kode < $%d`, argID)
		args = append(args, filter.Cursor)
		argID++
	}

	cols := `kode, tgl_entri, penerima, tipe_penerima, pesan, status, tgl_status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim`

	useTglLeading := filter.EndDate != nil && ((filter.PerintahProvider != nil && *filter.PerintahProvider) || (filter.ReplyToReseller != nil && *filter.ReplyToReseller))
	query, pqArgs := buildPageQueryPG(cols, "outbox", whereClause, args, argID, useTglLeading, filter.PageSize+1)
	rows, err := db.Query(ctx, query, pqArgs...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var outboxes []domain.Outbox

	for rows.Next() {
		var o domain.Outbox
		if err := rows.Scan(&o.Kode, &o.TglEntri, &o.Penerima, &o.TipePenerima, &o.Pesan, &o.Status, &o.TglStatus, &o.KodeInbox, &o.KodeTransaksi, &o.KodeReseller, &o.BebasBiaya, &o.IsPerintah, &o.KodeModul, &o.Prioritas, &o.ModulProses, &o.Pengirim, &o.KodeTerminal, &o.CtrKirim); err != nil {
			return nil, false, err
		}
		outboxes = append(outboxes, o)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasNextPage := len(outboxes) > filter.PageSize
	if hasNextPage {
		outboxes = outboxes[:filter.PageSize]
	}

	return outboxes, hasNextPage, nil
}

func (r *outboxPGRepository) Insert(ctx context.Context, tenant string, data domain.Outbox) error {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return err
	}
	query := `INSERT INTO outbox (tgl_entri, penerima, tipe_penerima, pesan, status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`
	_, err = db.Exec(ctx, query, time.Now(), data.Penerima, data.TipePenerima, data.Pesan, data.Status, data.KodeInbox, data.KodeTransaksi, data.KodeReseller, data.BebasBiaya, data.IsPerintah, data.KodeModul, data.Prioritas, data.ModulProses, data.Pengirim, data.KodeTerminal, data.CtrKirim)
	return err
}

func (r *outboxPGRepository) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateOutboxRequest) error {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return err
	}
	var tglStatus *time.Time
	if req.Status != nil {
		now := time.Now()
		tglStatus = &now
	}
	query := `UPDATE outbox SET
		tgl_status = COALESCE($1, tgl_status),
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
	_, err = db.Exec(ctx, query, tglStatus, req.Penerima, req.TipePenerima, req.Pesan, req.Status, req.BebasBiaya, req.KodeInbox, req.KodeTransaksi, req.KodeReseller, req.IsPerintah, req.KodeModul, req.Prioritas, req.ModulProses, req.Pengirim, req.KodeTerminal, req.CtrKirim, kode)
	return err
}

type outboxMSRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *outboxMSRepository) Get(ctx context.Context, tenant string, filter domain.OutboxFilter) ([]domain.Outbox, bool, error) {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return nil, false, err
	}

	whereClause, namedArgs := buildOutboxFilterMS(filter)

	if filter.LowerBound > 0 {
		whereClause += ` AND kode >= @lowerBound`
		namedArgs = append(namedArgs, sql.Named("lowerBound", filter.LowerBound))
	}

	if filter.Cursor > 0 {
		whereClause += ` AND kode < @cursor`
		namedArgs = append(namedArgs, sql.Named("cursor", filter.Cursor))
	}

	cols := `kode, tgl_entri, penerima, tipe_penerima, pesan, status, tgl_status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim`
	useTglLeading := filter.EndDate != nil && ((filter.PerintahProvider != nil && *filter.PerintahProvider) || (filter.ReplyToReseller != nil && *filter.ReplyToReseller))
	query, pqNamed := buildPageQueryMS(cols, "outbox", whereClause, namedArgs, useTglLeading, filter.PageSize+1)

	rows, err := db.QueryContext(ctx, query, pqNamed...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var outboxes []domain.Outbox

	for rows.Next() {
		var o domain.Outbox
		if err := rows.Scan(&o.Kode, &o.TglEntri, &o.Penerima, &o.TipePenerima, &o.Pesan, &o.Status, &o.TglStatus, &o.KodeInbox, &o.KodeTransaksi, &o.KodeReseller, &o.BebasBiaya, &o.IsPerintah, &o.KodeModul, &o.Prioritas, &o.ModulProses, &o.Pengirim, &o.KodeTerminal, &o.CtrKirim); err != nil {
			return nil, false, err
		}
		outboxes = append(outboxes, o)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasNextPage := len(outboxes) > filter.PageSize
	if hasNextPage {
		outboxes = outboxes[:filter.PageSize]
	}

	return outboxes, hasNextPage, nil
}

func (r *outboxMSRepository) Insert(ctx context.Context, tenant string, data domain.Outbox) error {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return err
	}
	query := `INSERT INTO outbox (tgl_entri, penerima, tipe_penerima, pesan, status, kode_inbox, kode_transaksi, kode_reseller, bebas_biaya, is_perintah, kode_modul, prioritas, modul_proses, pengirim, kode_terminal, ctr_kirim) VALUES (@tgl_entri, @penerima, @tipe_penerima, @pesan, @status, @kode_inbox, @kode_transaksi, @kode_reseller, @bebas_biaya, @is_perintah, @kode_modul, @prioritas, @modul_proses, @pengirim, @kode_terminal, @ctr_kirim)`
	_, err = db.ExecContext(ctx, query, sql.Named("tgl_entri", time.Now()), sql.Named("penerima", data.Penerima), sql.Named("tipe_penerima", data.TipePenerima), sql.Named("pesan", data.Pesan), sql.Named("status", data.Status), sql.Named("kode_inbox", data.KodeInbox), sql.Named("kode_transaksi", data.KodeTransaksi), sql.Named("kode_reseller", data.KodeReseller), sql.Named("bebas_biaya", data.BebasBiaya), sql.Named("is_perintah", data.IsPerintah), sql.Named("kode_modul", data.KodeModul), sql.Named("prioritas", data.Prioritas), sql.Named("modul_proses", data.ModulProses), sql.Named("pengirim", data.Pengirim), sql.Named("kode_terminal", data.KodeTerminal), sql.Named("ctr_kirim", data.CtrKirim))
	return err
}

func (r *outboxMSRepository) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateOutboxRequest) error {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return err
	}
	var tglStatus *time.Time
	if req.Status != nil {
		now := time.Now()
		tglStatus = &now
	}
	query := `UPDATE outbox SET
		tgl_status = COALESCE(@tgl_status, tgl_status),
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
	_, err = db.ExecContext(ctx, query,
		sql.Named("tgl_status", tglStatus),
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

func (r *outboxPGRepository) LowerBound(ctx context.Context, tenant string, filter domain.OutboxFilter) (int64, error) {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return 0, err
	}
	f := filter
	f.StartDate = nil
	whereClause, args, argID := buildOutboxFilterPG(f)
	if filter.EndDate != nil {
		cut, err := bisectCutPG(ctx, db, "outbox", *filter.EndDate)
		if err != nil {
			return 0, err
		}
		whereClause = " WHERE 1=1 AND kode <= $" + strconv.Itoa(argID) + whereClause[len(" WHERE 1=1"):]
		args = append(args, cut+bisectSlack)
		argID++
	}
	if filter.StartDate != nil {
		cut, err := bisectCutStartPG(ctx, db, "outbox", *filter.StartDate)
		if err != nil {
			return 0, err
		}
		whereClause += fmt.Sprintf(` AND kode > $%d`, argID)
		args = append(args, cut)
		argID++
	}
	query := `SELECT kode FROM outbox` + whereClause + fmt.Sprintf(` ORDER BY kode DESC LIMIT 1 OFFSET $%d`, argID)
	args = append(args, *filter.LimitTotal-1)
	var k int64
	err = db.QueryRow(ctx, query, args...).Scan(&k)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return k, nil
}

func (r *outboxMSRepository) LowerBound(ctx context.Context, tenant string, filter domain.OutboxFilter) (int64, error) {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return 0, err
	}
	whereClause, namedArgs := buildOutboxFilterMS(filter)
	if filter.EndDate != nil {
		cut, err := bisectCutMS(ctx, db, "outbox", *filter.EndDate)
		if err != nil {
			return 0, err
		}
		whereClause = " WHERE 1=1 AND kode <= @lowerCut" + whereClause[len(" WHERE 1=1"):]
		namedArgs = append(namedArgs, sql.Named("lowerCut", cut+bisectSlack))
	}
	namedArgs = append(namedArgs, sql.Named("lowerOffset", *filter.LimitTotal-1))
	query := `SELECT kode FROM outbox` + whereClause + ` ORDER BY kode DESC OFFSET @lowerOffset ROWS FETCH NEXT 1 ROWS ONLY`
	var k int64
	err = db.QueryRowContext(ctx, query, namedArgs...).Scan(&k)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return k, nil
}

func buildOutboxFilterPG(filter domain.OutboxFilter) (string, []any, int) {
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
		whereClause += fmt.Sprintf(` AND penerima ILIKE $%d`, argID)
		args = append(args, "%"+*filter.Penerima+"%")
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
	if filter.PerintahProvider != nil && *filter.PerintahProvider {
		whereClause += ` AND is_perintah = 1`
	} else if filter.ReplyToReseller != nil && *filter.ReplyToReseller {
		whereClause += ` AND kode_reseller IS NOT NULL AND is_perintah = 0`
	}

	return whereClause, args, argID
}

func buildOutboxFilterMS(filter domain.OutboxFilter) (string, []any) {
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
		whereClause += ` AND penerima LIKE '%' + @penerima + '%'`
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
	if filter.PerintahProvider != nil && *filter.PerintahProvider {
		whereClause += ` AND is_perintah = 1`
	} else if filter.ReplyToReseller != nil && *filter.ReplyToReseller {
		whereClause += ` AND kode_reseller IS NOT NULL AND is_perintah = 0`
	}

	return whereClause, namedArgs
}

func NewOutboxRepositories(dbRegistry *database.DBRegistry) *OutboxRepositories {
	return &OutboxRepositories{
		PG: &outboxPGRepository{dbRegistry: dbRegistry},
		MS: &outboxMSRepository{dbRegistry: dbRegistry},
	}
}
