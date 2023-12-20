package mainRepo

import (
	"commandos/internal/appconst"
	"commandos/internal/services/logger"
	"commandos/internal/types"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const USER_ROLE_SUPER_ADMIN = 2
const EVENT_SEVERITY_HIGH = 8
const NOTIF_STATUS_CREATED = 0

const cTable string = "commandos_commands"

type mainRepo struct {
	dbpool *pgxpool.Pool
	logger *logger.Logger
	ctx    context.Context
}

func NewSpoRepo(dbpool *pgxpool.Pool, logger *logger.Logger, ctx context.Context) *mainRepo {

	r := &mainRepo{
		dbpool,
		logger,
		ctx,
	}

	// Создаем таблицу, если она не существует
	r.createQueueTable()

	return r
}

func (r *mainRepo) createQueueTable() {

	sql := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		"id" serial PRIMARY KEY,
		"imei" varchar(45) NOT NULL,
		"name" varchar(45) NOT NULL,
		"params" jsonb NOT NULL,
		"author" varchar(45) NOT NULL,
		"dateon" timestamp NOT NULL default now(),
		"try_number" int not null default 0,
		"try_date" timestamp NULL,
		"response_date" timestamp,
		"response" jsonb,
		"raw_request" text,
		"raw_response" text,
        "abort_date" timestamp
	  )`, cTable)

	_, err := r.dbpool.Exec(r.ctx, sql)

	if err != nil {
		r.logger.Error.Fatalln(err)
		log.Fatal(err)
	}

	sql = fmt.Sprintf(`CREATE UNIQUE INDEX IF NOT EXISTS "%s_UNQ_IDX" ON "%s" USING btree ("imei", "name")
	WHERE "response_date" IS NULL AND "abort_date" IS NULL;`, cTable, cTable)

	_, err = r.dbpool.Exec(r.ctx, sql)

	if err != nil {
		r.logger.Error.Fatalln(err)
		log.Fatal(err)
	}
}

func (r *mainRepo) AddCommand(imei, name, params, author string) (int32, string, error) {
	qs := fmt.Sprintf(`INSERT INTO "%s" ("imei", "name", "params", "author") VALUES ($1, $2, $3, $4)`, cTable)

	_, err := r.dbpool.Exec(r.ctx, qs, imei, name, params, author)
	if err != nil {

		r.logger.Error.Println(err)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {

			switch pgErr.Code {
			case "23505":
				return appconst.DuplicateCommand, "Команда уже находится в очереди", nil
			}

			return appconst.UncaughtError, pgErr.Message, err
		}
		return appconst.UncaughtError, fmt.Sprint(err), err
	}
	return 0, "successfully", nil
}

func (r *mainRepo) ListCommands(imei string, offset int32, limit int32, newOnly bool) ([]types.Command, error) {

	newCond := ""

	dir := "DESC"

	if newOnly {
		newCond = `AND ("response_date" is null and "abort_date" is null)`
		dir = "ASC"
	}

	sql := fmt.Sprintf(`SELECT * from "%s" where "imei" = $1 %s ORDER BY dateon %s offset $2 limit $3`, cTable, newCond, dir)

	rows, err := r.dbpool.Query(r.ctx, sql, imei, offset, limit)

	if err != nil {
		r.logger.Error.Println(err)
		return []types.Command{}, err
	}

	defer rows.Close()

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Command])

	if err != nil && err != pgx.ErrNoRows {
		log.Println(err)
		return []types.Command{}, err
	}

	return items, nil
}

func (r *mainRepo) UpdCommand(id int32, responseDate time.Time, response, rawRequest, rawResponse string) (int32, int64, error) {

	nullableResponse := sql.NullString{String: response, Valid: len(response) > 0}
	nullableRawRequest := sql.NullString{String: rawRequest, Valid: len(rawRequest) > 0}
	nullableRawResponse := sql.NullString{String: rawResponse, Valid: len(rawResponse) > 0}

	qs := fmt.Sprintf(`UPDATE "%s" SET
                response_date=$2
                ,response=$3
                ,raw_request=$4
                ,raw_response=$5
                ,abort_date=null
            WHERE id=$1`, cTable)

	result, err := r.dbpool.Exec(r.ctx, qs, id, responseDate, nullableResponse, nullableRawRequest, nullableRawResponse)

	if err != nil {
		r.logger.Error.Println(err)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return appconst.UncaughtError, 0, pgErr
		}
		return appconst.UncaughtError, 0, err
	}

	return 0, result.RowsAffected(), nil
}

// UpdTry обновление попытки
func (r *mainRepo) UpdTry(id int32) error {

	qs := fmt.Sprintf(`UPDATE "%s" SET try_number = try_number + 1, try_date = now(), abort_date = case when try_number >= 2 then now() else null end WHERE id=$1`, cTable)

	_, err := r.dbpool.Exec(r.ctx, qs, id)

	return err
}
