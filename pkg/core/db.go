package core

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// DB represents the MySQL client
type DB interface {
	// Close closes the db connection.
	Close() error

	// ExecuteTransactionWithRetry wrapper for executing queries in a transaction and retries the transaction
	//  if transaction deadlock or transaction timeout.
	ExecuteTransactionWithRetry(
		ctx context.Context,
		maxRetries uint,
		delay,
		maxJitter time.Duration,
		errorMsg string,
		fn func(tx *sqlx.Tx) error) (err error)

	// Execute wrapper for executing the queries.
	Execute(fn func(conn *sqlx.DB) error) (err error)
}
