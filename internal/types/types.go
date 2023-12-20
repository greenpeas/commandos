package types

import "time"

type Command struct {
	Id           *int32     `db:"id"`
	Imei         string     `db:"imei"`
	Name         string     `db:"name"`
	Params       string     `db:"params"`
	Author       string     `db:"author"`
	Dateon       time.Time  `db:"dateon"`
	TryNumber    int32      `db:"try_number"`
	TryDate      *time.Time `db:"try_date"`
	ResponseDate *time.Time `db:"response_date"`
	Response     *string    `db:"response"`
	RawRequest   *string    `db:"raw_request"`
	RawResponse  *string    `db:"raw_response"`
	AbortDate    *time.Time `db:"abort_date"`
}

type CommandShort struct {
	Id     *int32 `db:"id"`
	Name   string `db:"name"`
	Params string `db:"params"`
}
