package worker

import (
	"context"
	"fmt"
	"insider-backend/internal/domain"
	"insider-backend/internal/repository"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// TransactionJob represents a transaction processing job
type TransactionJob struct {
	ID            string
	TransactionID uuid.UUID
	repositories  *repository.Repositories
}

// NewTransactionJob creates a new transaction job
func NewTransactionJob(transactionID uuid.UUID, repos *repository.Repositories) *TransactionJob {
	return &TransactionJob{
		ID:            fmt.Sprintf("transaction-%s", transactionID.String()),
		TransactionID: transactionID,
		repositories:  repos,
	}
}

// Execute processes the transaction
func (tj *TransactionJob) Execute(ctx context.Context) error {
	log.Info().
		Str("job_id", tj.ID).
		Str("transaction_id", tj.TransactionID.String()).
		Msg("Processing transaction")

	// Get the transaction
	transaction, err := tj.repositories.Transaction.GetByID(ctx, tj.TransactionID)
	if err != nil {
		return fmt.Errorf("failed to get transaction: %w", err)
	}

	// Check if transaction can be processed
	if !transaction.CanBeProcessed() {
		return fmt.Errorf("transaction %s cannot be processed, status: %s", transaction.ID, transaction.Status)
	}

	// Process based on transaction type
	switch transaction.Type {
	case domain.TransactionTypeCredit:
		return tj.processCredit(ctx, transaction)
	case domain.TransactionTypeDebit:
		return tj.processDebit(ctx, transaction)
	case domain.TransactionTypeTransfer:
		return tj.processTransfer(ctx, transaction)
	default:
		return fmt.Errorf("unknown transaction type: %s", transaction.Type)
	}
}

// GetID returns the job ID
func (tj *TransactionJob) GetID() string {
	return tj.ID
}

// GetType returns the job type
func (tj *TransactionJob) GetType() string {
	return "transaction"
}

// processCredit processes a credit transaction
func (tj *TransactionJob) processCredit(ctx context.Context, transaction *domain.Transaction) error {
	if transaction.ToUserID == nil {
		return fmt.Errorf("to_user_id is required for credit transaction")
	}

	// Get user balance
	balance, err := tj.repositories.Balance.GetByUserID(ctx, *transaction.ToUserID)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	previousAmount := balance.GetAmount()

	// Credit the amount
	if err := balance.Credit(transaction.Amount); err != nil {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("failed to credit balance: %w", err)
	}

	// Update balance in database
	if err := tj.repositories.Balance.UpdateWithLock(ctx, balance); err != nil {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Create balance history
	history := domain.NewBalanceHistory(*transaction.ToUserID, transaction.ID, balance.GetAmount(), previousAmount)
	if err := tj.repositories.Balance.CreateHistory(ctx, history); err != nil {
		log.Warn().Err(err).Msg("Failed to create balance history")
	}

	// Mark transaction as completed
	transaction.MarkCompleted()
	if err := tj.repositories.Transaction.Update(ctx, transaction); err != nil {
		log.Error().Err(err).Msg("Failed to update transaction status")
		return err
	}

	// Create audit log
	auditDetails := domain.BalanceAuditDetails{
		UserID:         *transaction.ToUserID,
		Amount:         balance.GetAmount(),
		PreviousAmount: previousAmount,
		TransactionID:  &transaction.ID,
		Operation:      "credit",
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeBalance,
		domain.ActionCredit,
		*transaction.ToUserID,
		auditDetails,
		nil,
		nil,
		"",
	)

	if err := tj.repositories.AuditLog.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("transaction_id", transaction.ID.String()).
		Float64("amount", transaction.Amount).
		Str("user_id", transaction.ToUserID.String()).
		Msg("Credit transaction completed")

	return nil
}

// processDebit processes a debit transaction
func (tj *TransactionJob) processDebit(ctx context.Context, transaction *domain.Transaction) error {
	if transaction.FromUserID == nil {
		return fmt.Errorf("from_user_id is required for debit transaction")
	}

	// Get user balance
	balance, err := tj.repositories.Balance.GetByUserID(ctx, *transaction.FromUserID)
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	previousAmount := balance.GetAmount()

	// Check if sufficient balance
	if !balance.HasSufficientBalance(transaction.Amount) {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", balance.GetAmount(), transaction.Amount)
	}

	// Debit the amount
	if err := balance.Debit(transaction.Amount); err != nil {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("failed to debit balance: %w", err)
	}

	// Update balance in database
	if err := tj.repositories.Balance.UpdateWithLock(ctx, balance); err != nil {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("failed to update balance: %w", err)
	}

	// Create balance history
	history := domain.NewBalanceHistory(*transaction.FromUserID, transaction.ID, balance.GetAmount(), previousAmount)
	if err := tj.repositories.Balance.CreateHistory(ctx, history); err != nil {
		log.Warn().Err(err).Msg("Failed to create balance history")
	}

	// Mark transaction as completed
	transaction.MarkCompleted()
	if err := tj.repositories.Transaction.Update(ctx, transaction); err != nil {
		log.Error().Err(err).Msg("Failed to update transaction status")
		return err
	}

	// Create audit log
	auditDetails := domain.BalanceAuditDetails{
		UserID:         *transaction.FromUserID,
		Amount:         balance.GetAmount(),
		PreviousAmount: previousAmount,
		TransactionID:  &transaction.ID,
		Operation:      "debit",
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeBalance,
		domain.ActionDebit,
		*transaction.FromUserID,
		auditDetails,
		nil,
		nil,
		"",
	)

	if err := tj.repositories.AuditLog.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("transaction_id", transaction.ID.String()).
		Float64("amount", transaction.Amount).
		Str("user_id", transaction.FromUserID.String()).
		Msg("Debit transaction completed")

	return nil
}

// processTransfer processes a transfer transaction
func (tj *TransactionJob) processTransfer(ctx context.Context, transaction *domain.Transaction) error {
	if transaction.FromUserID == nil || transaction.ToUserID == nil {
		return fmt.Errorf("both from_user_id and to_user_id are required for transfer transaction")
	}

	// Get both balances
	fromBalance, err := tj.repositories.Balance.GetByUserID(ctx, *transaction.FromUserID)
	if err != nil {
		return fmt.Errorf("failed to get from balance: %w", err)
	}

	toBalance, err := tj.repositories.Balance.GetByUserID(ctx, *transaction.ToUserID)
	if err != nil {
		return fmt.Errorf("failed to get to balance: %w", err)
	}

	// Check if sufficient balance
	if !fromBalance.HasSufficientBalance(transaction.Amount) {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("insufficient balance: have %.2f, need %.2f", fromBalance.GetAmount(), transaction.Amount)
	}

	previousFromAmount := fromBalance.GetAmount()
	previousToAmount := toBalance.GetAmount()

	// Debit from sender
	if err := fromBalance.Debit(transaction.Amount); err != nil {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("failed to debit from balance: %w", err)
	}

	// Credit to receiver
	if err := toBalance.Credit(transaction.Amount); err != nil {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("failed to credit to balance: %w", err)
	}

	// Update both balances atomically
	balances := []*domain.Balance{fromBalance, toBalance}
	if err := tj.repositories.Balance.BatchUpdate(ctx, balances); err != nil {
		transaction.MarkFailed()
		tj.repositories.Transaction.Update(ctx, transaction)
		return fmt.Errorf("failed to update balances: %w", err)
	}

	// Create balance histories
	fromHistory := domain.NewBalanceHistory(*transaction.FromUserID, transaction.ID, fromBalance.GetAmount(), previousFromAmount)
	toHistory := domain.NewBalanceHistory(*transaction.ToUserID, transaction.ID, toBalance.GetAmount(), previousToAmount)

	if err := tj.repositories.Balance.CreateHistory(ctx, fromHistory); err != nil {
		log.Warn().Err(err).Msg("Failed to create from balance history")
	}

	if err := tj.repositories.Balance.CreateHistory(ctx, toHistory); err != nil {
		log.Warn().Err(err).Msg("Failed to create to balance history")
	}

	// Mark transaction as completed
	transaction.MarkCompleted()
	if err := tj.repositories.Transaction.Update(ctx, transaction); err != nil {
		log.Error().Err(err).Msg("Failed to update transaction status")
		return err
	}

	// Create audit logs
	fromAuditDetails := domain.BalanceAuditDetails{
		UserID:         *transaction.FromUserID,
		Amount:         fromBalance.GetAmount(),
		PreviousAmount: previousFromAmount,
		TransactionID:  &transaction.ID,
		Operation:      "transfer_out",
	}

	toAuditDetails := domain.BalanceAuditDetails{
		UserID:         *transaction.ToUserID,
		Amount:         toBalance.GetAmount(),
		PreviousAmount: previousToAmount,
		TransactionID:  &transaction.ID,
		Operation:      "transfer_in",
	}

	fromAuditLog, _ := domain.NewAuditLog(
		domain.EntityTypeBalance,
		domain.ActionTransfer,
		*transaction.FromUserID,
		fromAuditDetails,
		nil,
		nil,
		"",
	)

	toAuditLog, _ := domain.NewAuditLog(
		domain.EntityTypeBalance,
		domain.ActionTransfer,
		*transaction.ToUserID,
		toAuditDetails,
		nil,
		nil,
		"",
	)

	if err := tj.repositories.AuditLog.Create(ctx, fromAuditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create from audit log")
	}

	if err := tj.repositories.AuditLog.Create(ctx, toAuditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create to audit log")
	}

	log.Info().
		Str("transaction_id", transaction.ID.String()).
		Float64("amount", transaction.Amount).
		Str("from_user_id", transaction.FromUserID.String()).
		Str("to_user_id", transaction.ToUserID.String()).
		Msg("Transfer transaction completed")

	return nil
}
