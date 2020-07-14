package tx

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/cenkalti/backoff/v4"
)

// Run runs fn in a database transaction.
// The context ctx is passed to fn, as well as the newly created
// transaction. If fn fails, it is repeated several times before
// giving up, with exponential backoff.
//
// There are a few rules that fn must respect:
//
// 1. fn must use the passed tx reference for all database calls.
// 2. fn must not commit or rollback the transaction: Run will do that.
// 3. fn must be idempotent, i.e. it may be called several times
//    without side effects.
//
// If fn returns nil, Run commits the transaction, returning
// the Commit and a nil error if it succeeds.
//
// If fn returns a non-nil value, Run rolls back the
// transaction and will return the reported error from fn.
//
// Run also recovers from panics, e.g. in fn.
func Run(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if rerr := recover(); rerr != nil {
			err = fmt.Errorf("%v", rerr)
			_ = tx.Rollback()
		}
	}()
	if err = fn(ctx, tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	_ = tx.Commit()
	return
}

// RunWithRetry is like Run but will retry
// several times with exponential backoff. In that case, fn must also
// be idempotent, i.e. it may be called several times without side effects.
func RunWithRetry(ctx context.Context, db *sql.DB, fn func(context.Context, *sql.Tx) error) (err error) {
	op := func() error {
		return Run(ctx, db, fn)
	}
	b := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
	return backoff.Retry(op, b)
}
