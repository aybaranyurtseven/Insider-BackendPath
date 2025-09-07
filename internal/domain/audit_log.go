package domain

import (
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	EntityType string          `json:"entity_type" db:"entity_type"`
	EntityID   uuid.UUID       `json:"entity_id" db:"entity_id"`
	Action     string          `json:"action" db:"action"`
	Details    json.RawMessage `json:"details" db:"details"`
	UserID     *uuid.UUID      `json:"user_id,omitempty" db:"user_id"`
	IPAddress  net.IP          `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  string          `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

type AuditLogFilter struct {
	EntityType string     `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty"`
	Action     string     `json:"action,omitempty"`
	UserID     *uuid.UUID `json:"user_id,omitempty"`
	FromDate   *time.Time `json:"from_date,omitempty"`
	ToDate     *time.Time `json:"to_date,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

const (
	// Entity types
	EntityTypeUser        = "user"
	EntityTypeTransaction = "transaction"
	EntityTypeBalance     = "balance"

	// Actions
	ActionCreate   = "create"
	ActionUpdate   = "update"
	ActionDelete   = "delete"
	ActionLogin    = "login"
	ActionLogout   = "logout"
	ActionCredit   = "credit"
	ActionDebit    = "debit"
	ActionTransfer = "transfer"
)

// NewAuditLog creates a new audit log entry
func NewAuditLog(entityType, action string, entityID uuid.UUID, details interface{}, userID *uuid.UUID, ipAddress net.IP, userAgent string) (*AuditLog, error) {
	var detailsJSON json.RawMessage
	var err error

	if details != nil {
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return nil, err
		}
	}

	return &AuditLog{
		ID:         uuid.New(),
		EntityType: entityType,
		EntityID:   entityID,
		Action:     action,
		Details:    detailsJSON,
		UserID:     userID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		CreatedAt:  time.Now(),
	}, nil
}

// GetDetailsAs unmarshals the details into the provided interface
func (al *AuditLog) GetDetailsAs(dest interface{}) error {
	if al.Details == nil {
		return nil
	}
	return json.Unmarshal(al.Details, dest)
}

// AuditLogBuilder helps build audit log entries
type AuditLogBuilder struct {
	auditLog *AuditLog
}

// NewAuditLogBuilder creates a new audit log builder
func NewAuditLogBuilder() *AuditLogBuilder {
	return &AuditLogBuilder{
		auditLog: &AuditLog{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
		},
	}
}

// WithEntity sets the entity type and ID
func (alb *AuditLogBuilder) WithEntity(entityType string, entityID uuid.UUID) *AuditLogBuilder {
	alb.auditLog.EntityType = entityType
	alb.auditLog.EntityID = entityID
	return alb
}

// WithAction sets the action
func (alb *AuditLogBuilder) WithAction(action string) *AuditLogBuilder {
	alb.auditLog.Action = action
	return alb
}

// WithDetails sets the details
func (alb *AuditLogBuilder) WithDetails(details interface{}) *AuditLogBuilder {
	if details != nil {
		if detailsJSON, err := json.Marshal(details); err == nil {
			alb.auditLog.Details = detailsJSON
		}
	}
	return alb
}

// WithUser sets the user information
func (alb *AuditLogBuilder) WithUser(userID *uuid.UUID) *AuditLogBuilder {
	alb.auditLog.UserID = userID
	return alb
}

// WithRequest sets the request information
func (alb *AuditLogBuilder) WithRequest(ipAddress net.IP, userAgent string) *AuditLogBuilder {
	alb.auditLog.IPAddress = ipAddress
	alb.auditLog.UserAgent = userAgent
	return alb
}

// Build returns the built audit log
func (alb *AuditLogBuilder) Build() *AuditLog {
	return alb.auditLog
}

// UserAuditDetails represents audit details for user operations
type UserAuditDetails struct {
	Username    string `json:"username,omitempty"`
	Email       string `json:"email,omitempty"`
	Role        string `json:"role,omitempty"`
	OldUsername string `json:"old_username,omitempty"`
	OldEmail    string `json:"old_email,omitempty"`
	OldRole     string `json:"old_role,omitempty"`
}

// TransactionAuditDetails represents audit details for transaction operations
type TransactionAuditDetails struct {
	FromUserID  *uuid.UUID `json:"from_user_id,omitempty"`
	ToUserID    *uuid.UUID `json:"to_user_id,omitempty"`
	Amount      float64    `json:"amount"`
	Type        string     `json:"type"`
	Status      string     `json:"status,omitempty"`
	OldStatus   string     `json:"old_status,omitempty"`
	Description string     `json:"description,omitempty"`
	ReferenceID string     `json:"reference_id,omitempty"`
}

// BalanceAuditDetails represents audit details for balance operations
type BalanceAuditDetails struct {
	UserID         uuid.UUID  `json:"user_id"`
	Amount         float64    `json:"amount"`
	PreviousAmount float64    `json:"previous_amount"`
	TransactionID  *uuid.UUID `json:"transaction_id,omitempty"`
	Operation      string     `json:"operation"`
}
