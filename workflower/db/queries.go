package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sazonovItas/mini-ci/workflower/db/gen/psql"
)

type Transactor interface {
	WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error
}

type txCtxKey struct{}

type Queries struct {
	pool *pgxpool.Pool
}

func NewQueries(pool *pgxpool.Pool) *Queries {
	return &Queries{pool: pool}
}

func (q *Queries) Queries(ctx context.Context) *psql.Queries {
	queries := psql.New(q.pool)

	tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx)
	if ok {
		return queries.WithTx(tx)
	}

	return queries
}

func (q *Queries) ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

func (q *Queries) Tx(ctx context.Context, begin bool) (tx pgx.Tx, found bool, err error) {
	tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx)
	if ok {
		return tx, true, nil
	}

	if begin {
		tx, err = q.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return nil, false, err
		}

		return tx, false, nil
	}

	return nil, false, nil
}

func (q *Queries) WithTx(ctx context.Context, txFunc func(txCtx context.Context) error) error {
	tx, found, err := q.Tx(ctx, true)
	if err != nil {
		return err
	}

	var txCtx context.Context
	if !found {
		defer func() {
			_ = tx.Rollback(ctx)
		}()

		txCtx = q.ContextWithTx(ctx, tx)
	}

	if err := txFunc(txCtx); err != nil {
		return err
	}

	if !found {
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}

	return nil
}
