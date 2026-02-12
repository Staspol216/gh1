package db

import (
	"context"

	pvz_ports "github.com/Staspol216/gh1/internal/ports"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type DB interface {
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error)
	ExecQueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row
}

type Database struct {
	tx pvz_ports.TransactionManager
}

func NewDatabase(tx pvz_ports.TransactionManager) *Database {
	return &Database{
		tx: tx,
	}
}

func (db Database) Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return pgxscan.Get(ctx, db.tx.GetQueryEngine(ctx), dest, query, args...)
}

func (db Database) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return pgxscan.Select(ctx, db.tx.GetQueryEngine(ctx), dest, query, args...)
}

func (db Database) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.tx.GetQueryEngine(ctx).Exec(ctx, query, args...)
}

func (db Database) ExecQueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return db.tx.GetQueryEngine(ctx).QueryRow(ctx, query, args...)
}
