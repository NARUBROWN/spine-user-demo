package interceptor

import (
	"errors"

	"github.com/NARUBROWN/spine/core"
	"github.com/uptrace/bun"
)

type TxInterceptor struct {
	db *bun.DB
}

func NewTxInterceptor(db *bun.DB) *TxInterceptor {
	return &TxInterceptor{db: db}
}

func (i *TxInterceptor) PreHandle(ctx core.ExecutionContext, meta core.HandlerMeta) error {
	reqCtx := ctx.Context()
	if reqCtx == nil {
		return errors.New("execution context has no request context")
	}

	tx, err := i.db.BeginTx(reqCtx, nil)
	if err != nil {
		return err
	}

	// ExecutionContext에 tx 저장
	ctx.Set("tx", tx)
	return nil
}

func (i *TxInterceptor) PostHandle(ctx core.ExecutionContext, meta core.HandlerMeta) {}

func (i *TxInterceptor) AfterCompletion(ctx core.ExecutionContext, meta core.HandlerMeta, err error) {
	v, ok := ctx.Get("tx")
	if !ok {
		return
	}

	tx, ok := v.(bun.Tx)
	if !ok {
		return
	}

	if err != nil {
		_ = tx.Rollback()
		return
	}

	_ = tx.Commit()
}
