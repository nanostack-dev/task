package task

import (
	"context"
	"database/sql"
)

type Querier interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func SetTransaction(tx *sql.Tx, ctx context.Context) context.Context {
	if tx == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, "tx", tx)
	return ctx
}

func GetTransaction(ctx context.Context) *sql.Tx {
	tx, _ := ctx.Value("tx").(*sql.Tx)
	return tx
}

func GetQueryOrTransaction(ctx context.Context, querier Querier) Querier {
	if tx := GetTransaction(ctx); tx != nil {
		return tx
	}
	return querier
}
