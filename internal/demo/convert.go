package demo

import "database/sql"

// boolToInt64 converts a Go bool to SQLite integer (1/0).
func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

// nullString converts a string to sql.NullString. Empty strings become NULL.
func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
