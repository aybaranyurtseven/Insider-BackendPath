package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"insider-backend/internal/domain"

	"github.com/google/uuid"
)

type BalanceRepository struct {
	db *sql.DB
}

func NewBalanceRepository(db *sql.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

func (r *BalanceRepository) Create(ctx context.Context, balance *domain.Balance) error {
	query := `
		INSERT INTO balances (user_id, amount, last_updated_at, version)
		VALUES ($1, $2, $3, $4)`

	_, err := r.db.ExecContext(ctx, query,
		balance.UserID,
		balance.Amount,
		balance.LastUpdatedAt,
		balance.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to create balance: %w", err)
	}

	return nil
}

func (r *BalanceRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Balance, error) {
	query := `
		SELECT user_id, amount, last_updated_at, version
		FROM balances WHERE user_id = $1`

	balance := domain.NewBalance(userID)
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&balance.UserID,
		&balance.Amount,
		&balance.LastUpdatedAt,
		&balance.Version,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Create new balance if it doesn't exist
			if createErr := r.Create(ctx, balance); createErr != nil {
				return nil, fmt.Errorf("failed to create new balance: %w", createErr)
			}
			return balance, nil
		}
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	return balance, nil
}

func (r *BalanceRepository) Update(ctx context.Context, balance *domain.Balance) error {
	query := `
		UPDATE balances 
		SET amount = $2, last_updated_at = $3, version = $4
		WHERE user_id = $1`

	result, err := r.db.ExecContext(ctx, query,
		balance.UserID,
		balance.Amount,
		balance.LastUpdatedAt,
		balance.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("balance not found")
	}

	return nil
}

func (r *BalanceRepository) UpdateWithLock(ctx context.Context, balance *domain.Balance) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Lock the row for update
	query := `
		SELECT user_id, amount, last_updated_at, version
		FROM balances WHERE user_id = $1 FOR UPDATE`

	currentBalance := &domain.Balance{}
	err = tx.QueryRowContext(ctx, query, balance.UserID).Scan(
		&currentBalance.UserID,
		&currentBalance.Amount,
		&currentBalance.LastUpdatedAt,
		&currentBalance.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to lock balance: %w", err)
	}

	// Check version for optimistic locking
	if currentBalance.Version != balance.Version-1 {
		return fmt.Errorf("balance version mismatch: expected %d, got %d", balance.Version-1, currentBalance.Version)
	}

	// Update the balance
	updateQuery := `
		UPDATE balances 
		SET amount = $2, last_updated_at = $3, version = $4
		WHERE user_id = $1`

	_, err = tx.ExecContext(ctx, updateQuery,
		balance.UserID,
		balance.Amount,
		balance.LastUpdatedAt,
		balance.Version,
	)

	if err != nil {
		return fmt.Errorf("failed to update balance: %w", err)
	}

	return tx.Commit()
}

func (r *BalanceRepository) BatchUpdate(ctx context.Context, balances []*domain.Balance) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		UPDATE balances 
		SET amount = $2, last_updated_at = $3, version = $4
		WHERE user_id = $1`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, balance := range balances {
		_, err = stmt.ExecContext(ctx,
			balance.UserID,
			balance.Amount,
			balance.LastUpdatedAt,
			balance.Version,
		)
		if err != nil {
			return fmt.Errorf("failed to update balance for user %s: %w", balance.UserID, err)
		}
	}

	return tx.Commit()
}

func (r *BalanceRepository) CreateHistory(ctx context.Context, history *domain.BalanceHistory) error {
	query := `
		INSERT INTO balance_history (id, user_id, amount, previous_amount, transaction_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.ExecContext(ctx, query,
		history.ID,
		history.UserID,
		history.Amount,
		history.PreviousAmount,
		history.TransactionID,
		history.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create balance history: %w", err)
	}

	return nil
}

func (r *BalanceRepository) GetHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.BalanceHistory, error) {
	query := `
		SELECT id, user_id, amount, previous_amount, transaction_id, created_at
		FROM balance_history 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance history: %w", err)
	}
	defer rows.Close()

	var histories []*domain.BalanceHistory
	for rows.Next() {
		history := &domain.BalanceHistory{}
		err := rows.Scan(
			&history.ID,
			&history.UserID,
			&history.Amount,
			&history.PreviousAmount,
			&history.TransactionID,
			&history.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan balance history: %w", err)
		}
		histories = append(histories, history)
	}

	return histories, nil
}

func (r *BalanceRepository) GetBalanceAtTime(ctx context.Context, userID uuid.UUID, timestamp string) (float64, error) {
	query := `
		SELECT amount
		FROM balance_history 
		WHERE user_id = $1 AND created_at <= $2
		ORDER BY created_at DESC
		LIMIT 1`

	var amount float64
	err := r.db.QueryRowContext(ctx, query, userID, timestamp).Scan(&amount)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil // No history found, return 0 balance
		}
		return 0, fmt.Errorf("failed to get balance at time: %w", err)
	}

	return amount, nil
}
