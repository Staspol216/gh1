package pvz_ports

import (
	"context"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type QueryEngine interface {
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

type TransactionManager interface {
	GetQueryEngine(ctxTx context.Context) QueryEngine
	RunReadCommitted(fn func(ctxTx context.Context) error) error
	RunRepeatableRead(fn func(ctxTx context.Context) error) error
	RunSerializable(fn func(ctxTx context.Context) error) error
}
