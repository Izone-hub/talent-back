package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"

	"github.com/Izone-hub/talent-backend/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func createAuditLogEntry(ctx context.Context, q *database.Queries, userID uuid.UUID, userRole string, action database.AuditAction, entityType string, entityID uuid.UUID, details any, oldValues any, newValues any) error {
	detailsJSON, err := marshalJSON(details)
	if err != nil {
		return fmt.Errorf("marshal audit details: %w", err)
	}
	oldJSON, err := marshalJSON(oldValues)
	if err != nil {
		return fmt.Errorf("marshal audit old values: %w", err)
	}
	newJSON, err := marshalJSON(newValues)
	if err != nil {
		return fmt.Errorf("marshal audit new values: %w", err)
	}

	_, err = q.CreateAuditLog(ctx, database.CreateAuditLogParams{
		UserID:       userID,
		UserEmail:    pgtype.Text{String: "", Valid: false},
		UserRole:     pgtype.Text{String: userRole, Valid: userRole != ""},
		Action:       action,
		EntityType:   pgtype.Text{String: entityType, Valid: true},
		EntityID:     entityID,
		IpAddress:    (*netip.Addr)(nil),
		UserAgent:    pgtype.Text{String: "", Valid: false},
		Details:      detailsJSON,
		OldValues:    oldJSON,
		NewValues:    newJSON,
		Status:       pgtype.Text{String: "success", Valid: true},
		ErrorMessage: pgtype.Text{String: "", Valid: false},
	})
	if err != nil {
		return fmt.Errorf("create audit log entry: %w", err)
	}
	return nil
}

func marshalJSON(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}
