package store

import "database/sql"

// ErrNotFound is returned when a queried resource does not exist.
var ErrNotFound = sql.ErrNoRows
