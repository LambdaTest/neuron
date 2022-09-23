package errors

import (
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

// ErrDupeKey is returned when a unique index prevents a value from being
// inserted or updated. CanRetry returns false on this error.
var ErrDupeKey = New("resource already exits")

// ErrDeadlock is returned when there is a transaction deadlock.
var ErrDeadlock = New("mysql transaction deadlock")

// ErrLockWaitTimeout is returned where there is a mysql lock wait timeout.
var ErrLockWaitTimeout = New("mysql lock wait timeout")

// ErrRowsNotFound is returned by Scan when QueryRow doesn't return a
// row.
var ErrRowsNotFound = sql.ErrNoRows

//  Error 1213: Deadlock found when trying to get lock; try restarting transaction"}
// ERROR 1205 (HY000): Lock wait timeout exceeded; try restarting transaction
const (
	mysqlDupEntryErrCode        = 1062
	mysqlDeadlockErrCode        = 1213
	mysqlLockWaitTimeoutErrCode = 1205
)

// SQLError returns an error in this package if possible. The error return value
// is an error in this package if the given error maps to one, else the given
// error is returned.
func SQLError(err error) error {
	mysqlErr, ok := err.(*mysql.MySQLError)
	if !ok {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRowsNotFound
		}
		return err
	}
	switch mysqlErr.Number {
	case mysqlDupEntryErrCode:
		return ErrDupeKey
	case mysqlDeadlockErrCode:
		return ErrDeadlock
	case mysqlLockWaitTimeoutErrCode:
		return ErrLockWaitTimeout
	}
	return mysqlErr
}
