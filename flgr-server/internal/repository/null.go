package repository

import "database/sql"

// nullString converts a nullable Go string pointer to a driver value.
func nullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// stringPtr converts a scanned nullable column back to a Go string pointer.
func stringPtr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}
