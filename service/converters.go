package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// uuidToPgUUID converts a uuid.UUID to a pgtype.UUID.
func uuidToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// pgUUIDToUUID converts a pgtype.UUID to a uuid.UUID.
func pgUUIDToUUID(id pgtype.UUID) uuid.UUID {
	if !id.Valid {
		return uuid.Nil
	}
	u, _ := uuid.FromBytes(id.Bytes[:])
	return u
}

// strPtrToPgText converts a *string to a pgtype.Text.
func strPtrToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

// strToPgText converts a string to a pgtype.Text, treating empty strings as NULL.
func strToPgText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// pgTextToStrPtr converts a pgtype.Text to a *string.
func pgTextToStrPtr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

// pgTextToString converts a pgtype.Text to a string.
func pgTextToString(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return t.String
}

// intPtrToPgInt4 converts an *int to a pgtype.Int4.
func intPtrToPgInt4(i *int) pgtype.Int4 {
	if i == nil {
		return pgtype.Int4{Valid: false}
	}
	return pgtype.Int4{Int32: int32(*i), Valid: true}
}

// intToPgInt4 converts an int to a pgtype.Int4.
func intToPgInt4(i int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(i), Valid: true}
}

// pgInt4ToIntPtr converts a pgtype.Int4 to an *int.
func pgInt4ToIntPtr(i pgtype.Int4) *int {
	if !i.Valid {
		return nil
	}
	v := int(i.Int32)
	return &v
}

// timePtrToPgTimestamp converts a *time.Time to a pgtype.Timestamp.
func timePtrToPgTimestamp(t *time.Time) pgtype.Timestamp {
	if t == nil {
		return pgtype.Timestamp{Valid: false}
	}
	return pgtype.Timestamp{Time: *t, Valid: true}
}

// pgTimestampToTime converts a pgtype.Timestamp to a time.Time.
func pgTimestampToTime(t pgtype.Timestamp) time.Time {
	return t.Time
}

// pgTimestampToTimePtr converts a pgtype.Timestamp to a *time.Time.
func pgTimestampToTimePtr(t pgtype.Timestamp) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}
