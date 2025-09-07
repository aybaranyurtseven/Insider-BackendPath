package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"insider-backend/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AuditLogRepository struct {
	db *sql.DB
}

func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, auditLog *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, entity_type, entity_id, action, details, user_id, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		auditLog.ID,
		auditLog.EntityType,
		auditLog.EntityID,
		auditLog.Action,
		auditLog.Details,
		auditLog.UserID,
		auditLog.IPAddress,
		auditLog.UserAgent,
		auditLog.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (r *AuditLogRepository) List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error) {
	query := `SELECT id, entity_type, entity_id, action, details, user_id, ip_address, user_agent, created_at FROM audit_logs`

	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.EntityType != "" {
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", argIndex))
		args = append(args, filter.EntityType)
		argIndex++
	}

	if filter.EntityID != nil {
		conditions = append(conditions, fmt.Sprintf("entity_id = $%d", argIndex))
		args = append(args, *filter.EntityID)
		argIndex++
	}

	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIndex))
		args = append(args, filter.Action)
		argIndex++
	}

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.FromDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.FromDate)
		argIndex++
	}

	if filter.ToDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.ToDate)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var auditLogs []*domain.AuditLog
	for rows.Next() {
		auditLog := &domain.AuditLog{}
		err := rows.Scan(
			&auditLog.ID,
			&auditLog.EntityType,
			&auditLog.EntityID,
			&auditLog.Action,
			&auditLog.Details,
			&auditLog.UserID,
			&auditLog.IPAddress,
			&auditLog.UserAgent,
			&auditLog.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, nil
}

func (r *AuditLogRepository) GetByEntityID(ctx context.Context, entityType string, entityID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
		SELECT id, entity_type, entity_id, action, details, user_id, ip_address, user_agent, created_at
		FROM audit_logs 
		WHERE entity_type = $1 AND entity_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, entityType, entityID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by entity ID: %w", err)
	}
	defer rows.Close()

	var auditLogs []*domain.AuditLog
	for rows.Next() {
		auditLog := &domain.AuditLog{}
		err := rows.Scan(
			&auditLog.ID,
			&auditLog.EntityType,
			&auditLog.EntityID,
			&auditLog.Action,
			&auditLog.Details,
			&auditLog.UserID,
			&auditLog.IPAddress,
			&auditLog.UserAgent,
			&auditLog.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, nil
}

func (r *AuditLogRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
		SELECT id, entity_type, entity_id, action, details, user_id, ip_address, user_agent, created_at
		FROM audit_logs 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit logs by user ID: %w", err)
	}
	defer rows.Close()

	var auditLogs []*domain.AuditLog
	for rows.Next() {
		auditLog := &domain.AuditLog{}
		err := rows.Scan(
			&auditLog.ID,
			&auditLog.EntityType,
			&auditLog.EntityID,
			&auditLog.Action,
			&auditLog.Details,
			&auditLog.UserID,
			&auditLog.IPAddress,
			&auditLog.UserAgent,
			&auditLog.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		auditLogs = append(auditLogs, auditLog)
	}

	return auditLogs, nil
}

func (r *AuditLogRepository) DeleteOlderThan(ctx context.Context, days int) error {
	query := `DELETE FROM audit_logs WHERE created_at < $1`

	cutoff := time.Now().AddDate(0, 0, -days)
	result, err := r.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("Deleted %d audit log entries older than %d days\n", rowsAffected, days)
	}

	return nil
}
