package domain

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Balance struct {
	UserID        uuid.UUID    `json:"user_id" db:"user_id"`
	Amount        float64      `json:"amount" db:"amount"`
	LastUpdatedAt time.Time    `json:"last_updated_at" db:"last_updated_at"`
	Version       int64        `json:"version" db:"version"`
	mu            sync.RWMutex `json:"-"`
}

type BalanceHistory struct {
	ID             uuid.UUID `json:"id" db:"id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Amount         float64   `json:"amount" db:"amount"`
	PreviousAmount float64   `json:"previous_amount" db:"previous_amount"`
	TransactionID  uuid.UUID `json:"transaction_id" db:"transaction_id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type BalanceSnapshot struct {
	UserID    uuid.UUID `json:"user_id"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
}

// NewBalance creates a new balance for a user
func NewBalance(userID uuid.UUID) *Balance {
	return &Balance{
		UserID:        userID,
		Amount:        0.0,
		LastUpdatedAt: time.Now(),
		Version:       1,
	}
}

// Credit adds amount to the balance (thread-safe)
func (b *Balance) Credit(amount float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if amount <= 0 {
		return fmt.Errorf("credit amount must be positive")
	}

	b.Amount += amount
	b.LastUpdatedAt = time.Now()
	b.Version++

	return nil
}

// Debit subtracts amount from the balance (thread-safe)
func (b *Balance) Debit(amount float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if amount <= 0 {
		return fmt.Errorf("debit amount must be positive")
	}

	if b.Amount < amount {
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", b.Amount, amount)
	}

	b.Amount -= amount
	b.LastUpdatedAt = time.Now()
	b.Version++

	return nil
}

// GetAmount returns the current balance amount (thread-safe)
func (b *Balance) GetAmount() float64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Amount
}

// HasSufficientBalance checks if balance is sufficient for the given amount (thread-safe)
func (b *Balance) HasSufficientBalance(amount float64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Amount >= amount
}

// GetSnapshot returns a snapshot of the current balance (thread-safe)
func (b *Balance) GetSnapshot() BalanceSnapshot {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return BalanceSnapshot{
		UserID:    b.UserID,
		Amount:    b.Amount,
		Timestamp: b.LastUpdatedAt,
	}
}

// SetAmount sets the balance amount directly (thread-safe) - use with caution
func (b *Balance) SetAmount(amount float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if amount < 0 {
		return fmt.Errorf("balance amount cannot be negative")
	}

	b.Amount = amount
	b.LastUpdatedAt = time.Now()
	b.Version++

	return nil
}

// Validate validates the balance
func (b *Balance) Validate() error {
	if b.Amount < 0 {
		return fmt.Errorf("balance amount cannot be negative")
	}
	return nil
}

// IsEmpty checks if the balance is zero
func (b *Balance) IsEmpty() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Amount == 0
}

// MarshalJSON customizes JSON marshaling (thread-safe)
func (b *Balance) MarshalJSON() ([]byte, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	type Alias Balance
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(b),
	})
}

// NewBalanceHistory creates a new balance history entry
func NewBalanceHistory(userID, transactionID uuid.UUID, newAmount, previousAmount float64) *BalanceHistory {
	return &BalanceHistory{
		ID:             uuid.New(),
		UserID:         userID,
		Amount:         newAmount,
		PreviousAmount: previousAmount,
		TransactionID:  transactionID,
		CreatedAt:      time.Now(),
	}
}

// BalanceOperation represents an atomic balance operation
type BalanceOperation struct {
	UserID    uuid.UUID
	Amount    float64
	Operation string // "credit" or "debit"
}

// BalanceBatch represents a batch of balance operations
type BalanceBatch struct {
	Operations    []BalanceOperation
	TransactionID uuid.UUID
	CreatedAt     time.Time
}

// NewBalanceBatch creates a new balance batch
func NewBalanceBatch(transactionID uuid.UUID, operations []BalanceOperation) *BalanceBatch {
	return &BalanceBatch{
		Operations:    operations,
		TransactionID: transactionID,
		CreatedAt:     time.Now(),
	}
}

// Validate validates the balance batch
func (bb *BalanceBatch) Validate() error {
	if len(bb.Operations) == 0 {
		return fmt.Errorf("balance batch cannot be empty")
	}

	for i, op := range bb.Operations {
		if op.Amount <= 0 {
			return fmt.Errorf("operation %d: amount must be positive", i)
		}
		if op.Operation != "credit" && op.Operation != "debit" {
			return fmt.Errorf("operation %d: invalid operation type %s", i, op.Operation)
		}
	}

	return nil
}
