package utils

import (
	"github.com/jackc/pgx/v5/pgtype"
)

// StringToText converts a string to pgtype.Text
func StringToText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// StringPtrToText converts a string pointer to pgtype.Text
func StringPtrToText(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// IntToInt4 converts an int to pgtype.Int4
func IntToInt4(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

// Int64ToInt8 converts an int64 to pgtype.Int8
func Int64ToInt8(i int64) pgtype.Int8 {
	return pgtype.Int8{Int64: i, Valid: true}
}

// BoolToBool converts a bool to pgtype.Bool
func BoolToBool(b bool) pgtype.Bool {
	return pgtype.Bool{Bool: b, Valid: true}
}

// GetTextValue safely gets string value from pgtype.Text
func GetTextValue(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}
