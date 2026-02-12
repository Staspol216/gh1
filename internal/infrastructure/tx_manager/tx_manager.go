package tx_manager

import (
	"context"

	pvz_ports "github.com/Staspol216/gh1/internal/ports"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type txManagerKey struct{}

var txKey = &txManagerKey{}

type TxManager struct {
	pool    *pgxpool.Pool
	context context.Context
}

func New(pool *pgxpool.Pool, context context.Context) *TxManager {
	return &TxManager{pool: pool, context: context}
}

func (m *TxManager) RunSerializable(fn func(ctxTx context.Context) error) error {
	opts := pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadWrite,
	}
	return m.beginFunc(opts, fn)
}

func (m *TxManager) RunRepeatableRead(fn func(ctxTx context.Context) error) error {
	opts := pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadWrite,
	}
	return m.beginFunc(opts, fn)
}

func (m *TxManager) RunReadCommitted(fn func(ctxTx context.Context) error) error {
	opts := pgx.TxOptions{
		IsoLevel:   pgx.ReadCommitted,
		AccessMode: pgx.ReadWrite,
	}
	return m.beginFunc(opts, fn)
}

func (m *TxManager) beginFunc(opts pgx.TxOptions, fn func(ctxTx context.Context) error) error {
	tx, err := m.pool.BeginTx(m.context, opts)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(m.context)
	}()

	ctxTx := context.WithValue(m.context, txKey, tx)
	if err := fn(ctxTx); err != nil {
		return err
	}

	return tx.Commit(m.context)
}

func (m *TxManager) GetQueryEngine(ctx context.Context) pvz_ports.QueryEngine {
	v, ok := ctx.Value(txKey).(pvz_ports.QueryEngine)
	if ok && v != nil {
		return v
	}

	return m.pool
}
