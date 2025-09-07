package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"insider-backend/internal/domain"
	"strings"

	"github.com/google/uuid"
)

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(ctx context.Context, transaction *domain.Transaction) error {
	query := `
		INSERT INTO transactions (id, from_user_id, to_user_id, amount, type, status, description, reference_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	_, err := r.db.ExecContext(ctx, query,
		transaction.ID,
		transaction.FromUserID,
		transaction.ToUserID,
		transaction.Amount,
		transaction.Type,
		transaction.Status,
		transaction.Description,
		transaction.ReferenceID,
		transaction.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, description, reference_id, created_at
		FROM transactions WHERE id = $1`

	transaction := &domain.Transaction{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&transaction.ID,
		&transaction.FromUserID,
		&transaction.ToUserID,
		&transaction.Amount,
		&transaction.Type,
		&transaction.Status,
		&transaction.Description,
		&transaction.ReferenceID,
		&transaction.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return transaction, nil
}

func (r *TransactionRepository) Update(ctx context.Context, transaction *domain.Transaction) error {
	query := `
		UPDATE transactions 
		SET status = $2, description = $3
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		transaction.ID,
		transaction.Status,
		transaction.Description,
	)

	if err != nil {
		return fmt.Errorf("failed to update transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}

func (r *TransactionRepository) List(ctx context.Context, filter domain.TransactionFilter) ([]*domain.Transaction, error) {
	query := `SELECT id, from_user_id, to_user_id, amount, type, status, description, reference_id, created_at FROM transactions`

	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("(from_user_id = $%d OR to_user_id = $%d)", argIndex, argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, *filter.Type)
		argIndex++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *filter.Status)
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
		return nil, fmt.Errorf("failed to list transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		transaction := &domain.Transaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.FromUserID,
			&transaction.ToUserID,
			&transaction.Amount,
			&transaction.Type,
			&transaction.Status,
			&transaction.Description,
			&transaction.ReferenceID,
			&transaction.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (r *TransactionRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, description, reference_id, created_at
		FROM transactions 
		WHERE from_user_id = $1 OR to_user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by user ID: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		transaction := &domain.Transaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.FromUserID,
			&transaction.ToUserID,
			&transaction.Amount,
			&transaction.Type,
			&transaction.Status,
			&transaction.Description,
			&transaction.ReferenceID,
			&transaction.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

func (r *TransactionRepository) GetByReferenceID(ctx context.Context, referenceID string) (*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, description, reference_id, created_at
		FROM transactions WHERE reference_id = $1`

	transaction := &domain.Transaction{}
	err := r.db.QueryRowContext(ctx, query, referenceID).Scan(
		&transaction.ID,
		&transaction.FromUserID,
		&transaction.ToUserID,
		&transaction.Amount,
		&transaction.Type,
		&transaction.Status,
		&transaction.Description,
		&transaction.ReferenceID,
		&transaction.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found")
		}
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	return transaction, nil
}

func (r *TransactionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TransactionStatus) error {
	query := `UPDATE transactions SET status = $2 WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id, status)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("transaction not found")
	}

	return nil
}

func (r *TransactionRepository) ListPending(ctx context.Context, limit int) ([]*domain.Transaction, error) {
	query := `
		SELECT id, from_user_id, to_user_id, amount, type, status, description, reference_id, created_at
		FROM transactions 
		WHERE status = 'pending'
		ORDER BY created_at ASC
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending transactions: %w", err)
	}
	defer rows.Close()

	var transactions []*domain.Transaction
	for rows.Next() {
		transaction := &domain.Transaction{}
		err := rows.Scan(
			&transaction.ID,
			&transaction.FromUserID,
			&transaction.ToUserID,
			&transaction.Amount,
			&transaction.Type,
			&transaction.Status,
			&transaction.Description,
			&transaction.ReferenceID,
			&transaction.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}
