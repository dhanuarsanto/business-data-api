package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.internal/business-data-api/internal/domain"
	"go.internal/business-data-api/internal/dto"
	"go.internal/business-data-api/pkg/database"
)

const bisectSlack = 50000

func bisectCutPG(ctx context.Context, db *pgxpool.Pool, table string, end time.Time) (int64, error) {
	var hi int64
	if err := db.QueryRow(ctx, "SELECT MAX(kode) FROM "+table).Scan(&hi); err != nil {
		return 0, err
	}
	if hi <= 0 {
		return 0, nil
	}
	lo := int64(0)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		var t time.Time
		err := db.QueryRow(ctx, "SELECT tgl_entri FROM "+table+" WHERE kode <= $1 ORDER BY kode DESC LIMIT 1", mid).Scan(&t)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				hi = mid - 1
				continue
			}
			return 0, err
		}
		if !t.After(end) {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo, nil
}

type InboxRepositories struct {
	PG domain.InboxRepository
	MS domain.InboxRepository
}

type inboxPGRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *inboxPGRepository) Get(ctx context.Context, tenant string, filter domain.InboxFilter) ([]domain.Inbox, bool, error) {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return nil, false, err
	}

	whereClause, args, argID := buildInboxFilterPG(filter)

	if filter.EndDate != nil {
		cut, err := bisectCutPG(ctx, db, "inbox", *filter.EndDate)
		if err != nil {
			return nil, false, err
		}
		whereClause = " WHERE 1=1 AND kode <= $" + strconv.Itoa(argID) + whereClause[len(" WHERE 1=1"):]
		args = append(args, cut+bisectSlack)
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

	cols := `kode, tgl_entri, tgl_status, pengirim, tipe_pengirim, penerima, pesan, status, kode_terminal, kode_reseller, kode_transaksi, is_jawaban, service_center, is_cs, kode_jawaban_cs, hash`

	useTglLeading := filter.EndDate != nil && ((filter.JawabanFromProvider != nil && *filter.JawabanFromProvider) || (filter.RequestFromReseller != nil && *filter.RequestFromReseller))
	query, pqArgs := buildPageQueryPG(cols, "inbox", whereClause, args, argID, useTglLeading, filter.PageSize+1)
	rows, err := db.Query(ctx, query, pqArgs...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var inboxes []domain.Inbox

	for rows.Next() {
		var i domain.Inbox
		if err := rows.Scan(&i.Kode, &i.TglEntri, &i.TglStatus, &i.Pengirim, &i.TipePengirim, &i.Penerima, &i.Pesan, &i.Status, &i.KodeTerminal, &i.KodeReseller, &i.KodeTransaksi, &i.IsJawaban, &i.ServiceCenter, &i.IsCs, &i.KodeJawabanCs, &i.Hash); err != nil {
			return nil, false, err
		}
		inboxes = append(inboxes, i)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasNextPage := len(inboxes) > filter.PageSize
	if hasNextPage {
		inboxes = inboxes[:filter.PageSize]
	}

	return inboxes, hasNextPage, nil
}

func (r *inboxPGRepository) Insert(ctx context.Context, tenant string, data domain.Inbox) error {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO inbox (tgl_entri, pengirim, tipe_pengirim, penerima, pesan, status, kode_terminal, kode_reseller, kode_transaksi, is_jawaban, service_center, is_cs, kode_jawaban_cs, hash) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`, time.Now(), data.Pengirim, data.TipePengirim, data.Penerima, data.Pesan, data.Status, data.KodeTerminal, data.KodeReseller, data.KodeTransaksi, data.IsJawaban, data.ServiceCenter, data.IsCs, data.KodeJawabanCs, data.Hash)
	return err
}

func (r *inboxPGRepository) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateInboxRequest) error {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return err
	}
	var tglStatus *time.Time
	if req.Status != nil {
		now := time.Now()
		tglStatus = &now
	}
	query := `UPDATE inbox SET
		tgl_status = COALESCE($1, tgl_status),
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
	_, err = db.Exec(ctx, query, tglStatus, req.Pesan, req.Status, req.Pengirim, req.TipePengirim, req.Penerima, req.KodeTerminal, req.KodeReseller, req.KodeTransaksi, req.IsJawaban, req.ServiceCenter, req.IsCs, req.KodeJawabanCs, req.Hash, kode)
	return err
}

type inboxMSRepository struct {
	dbRegistry *database.DBRegistry
}

func (r *inboxMSRepository) Get(ctx context.Context, tenant string, filter domain.InboxFilter) ([]domain.Inbox, bool, error) {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return nil, false, err
	}

	whereClause, namedArgs := buildInboxFilterMS(filter)

	if filter.LowerBound > 0 {
		whereClause += ` AND kode >= @lowerBound`
		namedArgs = append(namedArgs, sql.Named("lowerBound", filter.LowerBound))
	}

	if filter.Cursor > 0 {
		whereClause += ` AND kode < @cursor`
		namedArgs = append(namedArgs, sql.Named("cursor", filter.Cursor))
	}

	cols := `kode, tgl_entri, tgl_status, pengirim, tipe_pengirim, penerima, pesan, status, kode_terminal, kode_reseller, kode_transaksi, is_jawaban, service_center, is_cs, kode_jawaban_cs, hash`
	useTglLeading := filter.EndDate != nil && ((filter.JawabanFromProvider != nil && *filter.JawabanFromProvider) || (filter.RequestFromReseller != nil && *filter.RequestFromReseller))
	query, pqNamed := buildPageQueryMS(cols, "inbox", whereClause, namedArgs, useTglLeading, filter.PageSize+1)

	rows, err := db.QueryContext(ctx, query, pqNamed...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var inboxes []domain.Inbox

	for rows.Next() {
		var i domain.Inbox
		if err := rows.Scan(&i.Kode, &i.TglEntri, &i.TglStatus, &i.Pengirim, &i.TipePengirim, &i.Penerima, &i.Pesan, &i.Status, &i.KodeTerminal, &i.KodeReseller, &i.KodeTransaksi, &i.IsJawaban, &i.ServiceCenter, &i.IsCs, &i.KodeJawabanCs, &i.Hash); err != nil {
			return nil, false, err
		}
		inboxes = append(inboxes, i)
	}

	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	hasNextPage := len(inboxes) > filter.PageSize
	if hasNextPage {
		inboxes = inboxes[:filter.PageSize]
	}

	return inboxes, hasNextPage, nil
}

func (r *inboxMSRepository) Insert(ctx context.Context, tenant string, data domain.Inbox) error {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO inbox (tgl_entri, pengirim, tipe_pengirim, penerima, pesan, status, kode_terminal, kode_reseller, kode_transaksi, is_jawaban, service_center, is_cs, kode_jawaban_cs, hash) VALUES (@tgl_entri, @pengirim, @tipe_pengirim, @penerima, @pesan, @status, @kode_terminal, @kode_reseller, @kode_transaksi, @is_jawaban, @service_center, @is_cs, @kode_jawaban_cs, @hash)`, sql.Named("tgl_entri", time.Now()), sql.Named("pengirim", data.Pengirim), sql.Named("tipe_pengirim", data.TipePengirim), sql.Named("penerima", data.Penerima), sql.Named("pesan", data.Pesan), sql.Named("status", data.Status), sql.Named("kode_terminal", data.KodeTerminal), sql.Named("kode_reseller", data.KodeReseller), sql.Named("kode_transaksi", data.KodeTransaksi), sql.Named("is_jawaban", data.IsJawaban), sql.Named("service_center", data.ServiceCenter), sql.Named("is_cs", data.IsCs), sql.Named("kode_jawaban_cs", data.KodeJawabanCs), sql.Named("hash", data.Hash))
	return err
}

func (r *inboxMSRepository) Update(ctx context.Context, tenant string, kode int64, req dto.UpdateInboxRequest) error {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return err
	}
	var tglStatus *time.Time
	if req.Status != nil {
		now := time.Now()
		tglStatus = &now
	}
	query := `UPDATE inbox SET
		tgl_status = COALESCE(@tgl_status, tgl_status),
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
	_, err = db.ExecContext(ctx, query,
		sql.Named("tgl_status", tglStatus),
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

func (r *inboxPGRepository) LowerBound(ctx context.Context, tenant string, filter domain.InboxFilter) (int64, error) {
	db, err := r.dbRegistry.Postgres(tenant)
	if err != nil {
		return 0, err
	}
	whereClause, args, _ := buildInboxFilterPG(filter)
	if filter.EndDate != nil {
		cut, err := bisectCutPG(ctx, db, "inbox", *filter.EndDate)
		if err != nil {
			return 0, err
		}
		whereClause = " WHERE 1=1 AND kode <= $" + strconv.Itoa(len(args)+1) + whereClause[len(" WHERE 1=1"):]
		args = append(args, cut+bisectSlack)
	}
	query := `SELECT kode FROM inbox` + whereClause + fmt.Sprintf(` ORDER BY kode DESC LIMIT 1 OFFSET $%d`, len(args)+1)
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

func bisectCutMS(ctx context.Context, db *sql.DB, table string, end time.Time) (int64, error) {
	var hi int64
	if err := db.QueryRowContext(ctx, "SELECT MAX(kode) FROM "+table).Scan(&hi); err != nil {
		return 0, err
	}
	if hi <= 0 {
		return 0, nil
	}
	lo := int64(0)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		var t time.Time
		err := db.QueryRowContext(ctx, "SELECT TOP (1) tgl_entri FROM "+table+" WHERE kode <= ? ORDER BY kode DESC", mid).Scan(&t)
		if err != nil {
			if err == sql.ErrNoRows {
				hi = mid - 1
				continue
			}
			return 0, err
		}
		if !t.After(end) {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo, nil
}

func (r *inboxMSRepository) LowerBound(ctx context.Context, tenant string, filter domain.InboxFilter) (int64, error) {
	db, err := r.dbRegistry.MSSQL(tenant)
	if err != nil {
		return 0, err
	}
	whereClause, namedArgs := buildInboxFilterMS(filter)
	if filter.EndDate != nil {
		cut, err := bisectCutMS(ctx, db, "inbox", *filter.EndDate)
		if err != nil {
			return 0, err
		}
		whereClause = " WHERE 1=1 AND kode <= @lowerCut" + whereClause[len(" WHERE 1=1"):]
		namedArgs = append(namedArgs, sql.Named("lowerCut", cut+bisectSlack))
	}
	namedArgs = append(namedArgs, sql.Named("lowerOffset", *filter.LimitTotal-1))
	query := `SELECT kode FROM inbox` + whereClause + ` ORDER BY kode DESC OFFSET @lowerOffset ROWS FETCH NEXT 1 ROWS ONLY`
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

func buildInboxFilterPG(filter domain.InboxFilter) (string, []any, int) {
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
		whereClause += fmt.Sprintf(` AND pengirim ILIKE $%d`, argID)
		args = append(args, "%"+*filter.Pengirim+"%")
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
	if filter.JawabanFromProvider != nil && *filter.JawabanFromProvider {
		whereClause += ` AND is_jawaban = 1`
	} else if filter.RequestFromReseller != nil && *filter.RequestFromReseller {
		whereClause += ` AND kode_reseller IS NOT NULL AND is_jawaban = 0`
	}

	return whereClause, args, argID
}

func buildInboxFilterMS(filter domain.InboxFilter) (string, []any) {
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
		whereClause += ` AND pengirim LIKE '%' + @pengirim + '%'`
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
	if filter.JawabanFromProvider != nil && *filter.JawabanFromProvider {
		whereClause += ` AND is_jawaban = 1`
	} else if filter.RequestFromReseller != nil && *filter.RequestFromReseller {
		whereClause += ` AND kode_reseller IS NOT NULL AND is_jawaban = 0`
	}

	return whereClause, namedArgs
}

func NewInboxRepositories(dbRegistry *database.DBRegistry) *InboxRepositories {
	return &InboxRepositories{
		PG: &inboxPGRepository{dbRegistry: dbRegistry},
		MS: &inboxMSRepository{dbRegistry: dbRegistry},
	}
}
