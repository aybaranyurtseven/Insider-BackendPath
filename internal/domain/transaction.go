package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type TransactionType string
type TransactionStatus string

const (
	TransactionTypeCredit   TransactionType = "credit"
	TransactionTypeDebit    TransactionType = "debit"
	TransactionTypeTransfer TransactionType = "transfer"
)

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

type Transaction struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	FromUserID  *uuid.UUID        `json:"from_user_id,omitempty" db:"from_user_id"`
	ToUserID    *uuid.UUID        `json:"to_user_id,omitempty" db:"to_user_id"`
	Amount      float64           `json:"amount" db:"amount"`
	Type        TransactionType   `json:"type" db:"type"`
	Status      TransactionStatus `json:"status" db:"status"`
	Description string            `json:"description,omitempty" db:"description"`
	ReferenceID string            `json:"reference_id,omitempty" db:"reference_id"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
}

type CreateTransactionRequest struct {
	FromUserID  *uuid.UUID `json:"from_user_id,omitempty"`
	ToUserID    *uuid.UUID `json:"to_user_id,omitempty"`
	Amount      float64    `json:"amount"`
	Type        string     `json:"type"`
	Description string     `json:"description,omitempty"`
	ReferenceID string     `json:"reference_id,omitempty"`
}

type TransactionFilter struct {
	UserID   *uuid.UUID         `json:"user_id,omitempty"`
	Type     *TransactionType   `json:"type,omitempty"`
	Status   *TransactionStatus `json:"status,omitempty"`
	FromDate *time.Time         `json:"from_date,omitempty"`
	ToDate   *time.Time         `json:"to_date,omitempty"`
	Limit    int                `json:"limit,omitempty"`
	Offset   int                `json:"offset,omitempty"`
}

// NewTransaction creates a new transaction with validation
func NewTransaction(fromUserID, toUserID *uuid.UUID, amount float64, txType TransactionType, description, referenceID string) (*Transaction, error) {
	transaction := &Transaction{
		ID:          uuid.New(),
		FromUserID:  fromUserID,
		ToUserID:    toUserID,
		Amount:      amount,
		Type:        txType,
		Status:      TransactionStatusPending,
		Description: description,
		ReferenceID: referenceID,
		CreatedAt:   time.Now(),
	}

	if err := transaction.Validate(); err != nil {
		return nil, err
	}

	return transaction, nil
}

// Validate validates transaction fields
func (t *Transaction) Validate() error {
	if t.Amount <= 0 {
		return fmt.Errorf("amount must be greater than 0")
	}

	switch t.Type {
	case TransactionTypeCredit:
		if t.ToUserID == nil {
			return fmt.Errorf("to_user_id is required for credit transactions")
		}
		if t.FromUserID != nil {
			return fmt.Errorf("from_user_id should be nil for credit transactions")
		}
	case TransactionTypeDebit:
		if t.FromUserID == nil {
			return fmt.Errorf("from_user_id is required for debit transactions")
		}
		if t.ToUserID != nil {
			return fmt.Errorf("to_user_id should be nil for debit transactions")
		}
	case TransactionTypeTransfer:
		if t.FromUserID == nil || t.ToUserID == nil {
			return fmt.Errorf("both from_user_id and to_user_id are required for transfer transactions")
		}
		if *t.FromUserID == *t.ToUserID {
			return fmt.Errorf("from_user_id and to_user_id cannot be the same")
		}
	default:
		return fmt.Errorf("invalid transaction type: %s", t.Type)
	}

	return nil
}

// MarkCompleted marks the transaction as completed
func (t *Transaction) MarkCompleted() {
	t.Status = TransactionStatusCompleted
}

// MarkFailed marks the transaction as failed
func (t *Transaction) MarkFailed() {
	t.Status = TransactionStatusFailed
}

// MarkCancelled marks the transaction as cancelled
func (t *Transaction) MarkCancelled() {
	t.Status = TransactionStatusCancelled
}

// IsCompleted checks if the transaction is completed
func (t *Transaction) IsCompleted() bool {
	return t.Status == TransactionStatusCompleted
}

// IsPending checks if the transaction is pending
func (t *Transaction) IsPending() bool {
	return t.Status == TransactionStatusPending
}

// CanBeProcessed checks if the transaction can be processed
func (t *Transaction) CanBeProcessed() bool {
	return t.Status == TransactionStatusPending
}

// GetAffectedUserIDs returns the list of user IDs affected by this transaction
func (t *Transaction) GetAffectedUserIDs() []uuid.UUID {
	var userIDs []uuid.UUID

	if t.FromUserID != nil {
		userIDs = append(userIDs, *t.FromUserID)
	}

	if t.ToUserID != nil {
		userIDs = append(userIDs, *t.ToUserID)
	}

	return userIDs
}

// MarshalJSON customizes JSON marshaling
func (t *Transaction) MarshalJSON() ([]byte, error) {
	type Alias Transaction
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(t),
	})
}

func IsValidTransactionType(txType string) bool {
	validTypes := []string{
		string(TransactionTypeCredit),
		string(TransactionTypeDebit),
		string(TransactionTypeTransfer),
	}

	for _, validType := range validTypes {
		if txType == validType {
			return true
		}
	}
	return false
}

func IsValidTransactionStatus(status string) bool {
	validStatuses := []string{
		string(TransactionStatusPending),
		string(TransactionStatusCompleted),
		string(TransactionStatusFailed),
		string(TransactionStatusCancelled),
	}

	for _, validStatus := range validStatuses {
		if status == validStatus {
			return true
		}
	}
	return false
}
