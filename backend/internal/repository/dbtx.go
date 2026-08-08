package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arg ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, arg ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, arg ...any) pgx.Row
}
