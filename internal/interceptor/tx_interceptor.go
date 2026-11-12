package interceptor

import (
	"errors"

	"github.com/NARUBROWN/spine/core"
	spinectx "github.com/NARUBROWN/spine/pkg/spine"
	"github.com/uptrace/bun"
)

type TxInterceptor struct {
	db *bun.DB
}

func (*TxInterceptor) Constructor(db *bun.DB) *TxInterceptor {
	return &TxInterceptor{db: db}
}

func (i *TxInterceptor) PreHandle(ctx spinectx.Ctx, meta core.HandlerMeta) error {
	reqCtx := ctx.ExecutionContext().Context()
	if reqCtx == nil {
		return errors.New("execution context has no request context")
	}

	tx, err := i.db.BeginTx(reqCtx, nil)
	if err != nil {
		return err
	}

	// ExecutionContext에 tx 저장
	ctx.ExecutionContext().Set("tx", tx)
	return nil
}

func (i *TxInterceptor) PostHandle(ctx spinectx.Ctx, meta core.HandlerMeta) {}

func (i *TxInterceptor) BeforeResponse(
	ctx spinectx.Ctx,
	meta core.HandlerMeta,
	executionErr error,
) error {
	v, ok := ctx.ExecutionContext().Get("tx")
	if !ok {
		return nil
	}

	tx, ok := v.(bun.Tx)
	if !ok {
		return errors.New("execution context has an invalid transaction")
	}

	if executionErr != nil {
		return tx.Rollback()
	}

	return tx.Commit()
}

func (i *TxInterceptor) AfterCompletion(
	ctx spinectx.Ctx,
	meta core.HandlerMeta,
	err error,
) {
}
