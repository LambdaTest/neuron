package db

import (
	"context"
	"errors"
	"time"

	errs "github.com/LambdaTest/neuron/pkg/errors"
	"github.com/LambdaTest/neuron/pkg/lumber"
	"github.com/avast/retry-go/v4"
	"github.com/jmoiron/sqlx"
)

//TODO: store timestamp to atleast ms precision

// DB is a pool of zero or more underlying connections to
// the neuron database.
type DB struct {
	conn   *sqlx.DB
	logger lumber.Logger
}

// Execute and executes a function. Any error that is returned from the function is returned
// from the Execute() method.
func (db *DB) Execute(fn func(conn *sqlx.DB) error) (err error) {
	err = fn(db.conn)
	return err
}

// ExecuteTransactionWithRetry wrapper for executing queries in a transaction and retries the transaction
// if transaction deadlock or transaction timeout.
func (db *DB) ExecuteTransactionWithRetry(
	ctx context.Context,
	maxRetries uint,
	delay,
	maxJitter time.Duration,
	errorMsg string,
	fn func(tx *sqlx.Tx) error) (err error) {
	return retry.Do(func() error {
		return db.executeTransaction(ctx, fn)
	}, retry.Context(ctx),
		retry.LastErrorOnly(true),
		retry.Attempts(maxRetries),
		retry.Delay(delay),
		retry.MaxJitter(maxJitter),
		retry.RetryIf(func(err error) bool {
			parseErr := errs.SQLError(err)
			// retry in case of deadlock or lock wait timeout
			if errors.Is(parseErr, errs.ErrDeadlock) || errors.Is(parseErr, errs.ErrLockWaitTimeout) {
				return true
			}
			return false
		}),
		retry.OnRetry(func(n uint, err error) {
			db.logger.Errorf("%s, retry %d, error: %+v", errorMsg, n, err)
		}),
	)
}

// executeTransaction executes a function within the context of a read-write managed
// transaction. If no error is returned from the function then the
// transaction is committed. If an error is returned then the entire
// transaction is rolled back. Any error that is returned from the function
// or returned from the commit is returned from the executeTransaction() method.
func (db *DB) executeTransaction(ctx context.Context, fn func(tx *sqlx.Tx) error) (err error) {
	tx, err := db.conn.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			// a panic occurred, rollback and repanic
			if rerr := tx.Rollback(); rerr != nil {
				db.logger.Errorf("error while performing rollback, %v", rerr)
			}
			db.logger.Errorf("panic while executing query: %+v\n", p)
			panic(p) // More often than not a panic is due to a programming error, and should be corrected
		} else if err != nil {
			// something went wrong, rollback. In case of context cancelation or timeout,
			// tx is automatically rolled back by underlying package. See db.conn.BeginTxx()
			if err != context.Canceled && err != context.DeadlineExceeded {
				if rerr := tx.Rollback(); rerr != nil {
					db.logger.Errorf("error while performing rollback, %v", rerr)
				}
			}
		} else {
			// all good, commit
			err = tx.Commit()
		}
	}()
	err = fn(tx)
	return err
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
