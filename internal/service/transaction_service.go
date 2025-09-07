package service

import (
	"context"
	"fmt"
	"insider-backend/internal/domain"
	"insider-backend/internal/repository"
	"insider-backend/internal/worker"
	"net"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type TransactionService struct {
	transactionRepo repository.TransactionRepository
	balanceRepo     repository.BalanceRepository
	userRepo        repository.UserRepository
	auditRepo       repository.AuditLogRepository
	cacheRepo       repository.CacheRepository
	workerPool      *worker.WorkerPool
}

func NewTransactionService(repos *repository.Repositories, workerPool *worker.WorkerPool) *TransactionService {
	return &TransactionService{
		transactionRepo: repos.Transaction,
		balanceRepo:     repos.Balance,
		userRepo:        repos.User,
		auditRepo:       repos.AuditLog,
		cacheRepo:       repos.Cache,
		workerPool:      workerPool,
	}
}

// CreateCredit creates a credit transaction
func (s *TransactionService) CreateCredit(ctx context.Context, req domain.CreateTransactionRequest, userID *uuid.UUID, ipAddress net.IP, userAgent string) (*domain.Transaction, error) {
	log.Info().
		Str("to_user_id", req.ToUserID.String()).
		Float64("amount", req.Amount).
		Msg("Creating credit transaction")

	if req.ToUserID == nil {
		return nil, fmt.Errorf("to_user_id is required for credit transaction")
	}

	// Verify target user exists
	_, err := s.userRepo.GetByID(ctx, *req.ToUserID)
	if err != nil {
		return nil, fmt.Errorf("target user not found: %w", err)
	}

	// Create transaction
	transaction, err := domain.NewTransaction(
		nil,
		req.ToUserID,
		req.Amount,
		domain.TransactionTypeCredit,
		req.Description,
		req.ReferenceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Save transaction
	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	// Submit to worker pool for processing
	job := worker.NewTransactionJob(transaction.ID, &repository.Repositories{
		Transaction: s.transactionRepo,
		Balance:     s.balanceRepo,
		User:        s.userRepo,
		AuditLog:    s.auditRepo,
		Cache:       s.cacheRepo,
	})

	if err := s.workerPool.SubmitJob(job); err != nil {
		log.Error().Err(err).Str("transaction_id", transaction.ID.String()).Msg("Failed to submit transaction job")
		// Mark transaction as failed
		transaction.MarkFailed()
		s.transactionRepo.Update(ctx, transaction)
		return nil, fmt.Errorf("failed to process transaction: %w", err)
	}

	// Create audit log
	auditDetails := domain.TransactionAuditDetails{
		ToUserID:    req.ToUserID,
		Amount:      req.Amount,
		Type:        string(domain.TransactionTypeCredit),
		Status:      string(transaction.Status),
		Description: req.Description,
		ReferenceID: req.ReferenceID,
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeTransaction,
		domain.ActionCreate,
		transaction.ID,
		auditDetails,
		userID,
		ipAddress,
		userAgent,
	)

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("transaction_id", transaction.ID.String()).
		Float64("amount", req.Amount).
		Msg("Credit transaction created")

	return transaction, nil
}

// CreateDebit creates a debit transaction
func (s *TransactionService) CreateDebit(ctx context.Context, req domain.CreateTransactionRequest, userID *uuid.UUID, ipAddress net.IP, userAgent string) (*domain.Transaction, error) {
	log.Info().
		Str("from_user_id", req.FromUserID.String()).
		Float64("amount", req.Amount).
		Msg("Creating debit transaction")

	if req.FromUserID == nil {
		return nil, fmt.Errorf("from_user_id is required for debit transaction")
	}

	// Verify source user exists
	_, err := s.userRepo.GetByID(ctx, *req.FromUserID)
	if err != nil {
		return nil, fmt.Errorf("source user not found: %w", err)
	}

	// Check balance before creating transaction
	balance, err := s.balanceRepo.GetByUserID(ctx, *req.FromUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if !balance.HasSufficientBalance(req.Amount) {
		return nil, fmt.Errorf("insufficient balance: have %.2f, need %.2f", balance.GetAmount(), req.Amount)
	}

	// Create transaction
	transaction, err := domain.NewTransaction(
		req.FromUserID,
		nil,
		req.Amount,
		domain.TransactionTypeDebit,
		req.Description,
		req.ReferenceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Save transaction
	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	// Submit to worker pool for processing
	job := worker.NewTransactionJob(transaction.ID, &repository.Repositories{
		Transaction: s.transactionRepo,
		Balance:     s.balanceRepo,
		User:        s.userRepo,
		AuditLog:    s.auditRepo,
		Cache:       s.cacheRepo,
	})

	if err := s.workerPool.SubmitJob(job); err != nil {
		log.Error().Err(err).Str("transaction_id", transaction.ID.String()).Msg("Failed to submit transaction job")
		// Mark transaction as failed
		transaction.MarkFailed()
		s.transactionRepo.Update(ctx, transaction)
		return nil, fmt.Errorf("failed to process transaction: %w", err)
	}

	// Create audit log
	auditDetails := domain.TransactionAuditDetails{
		FromUserID:  req.FromUserID,
		Amount:      req.Amount,
		Type:        string(domain.TransactionTypeDebit),
		Status:      string(transaction.Status),
		Description: req.Description,
		ReferenceID: req.ReferenceID,
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeTransaction,
		domain.ActionCreate,
		transaction.ID,
		auditDetails,
		userID,
		ipAddress,
		userAgent,
	)

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("transaction_id", transaction.ID.String()).
		Float64("amount", req.Amount).
		Msg("Debit transaction created")

	return transaction, nil
}

// CreateTransfer creates a transfer transaction
func (s *TransactionService) CreateTransfer(ctx context.Context, req domain.CreateTransactionRequest, userID *uuid.UUID, ipAddress net.IP, userAgent string) (*domain.Transaction, error) {
	log.Info().
		Str("from_user_id", req.FromUserID.String()).
		Str("to_user_id", req.ToUserID.String()).
		Float64("amount", req.Amount).
		Msg("Creating transfer transaction")

	if req.FromUserID == nil || req.ToUserID == nil {
		return nil, fmt.Errorf("both from_user_id and to_user_id are required for transfer transaction")
	}

	if *req.FromUserID == *req.ToUserID {
		return nil, fmt.Errorf("cannot transfer to the same user")
	}

	// Verify both users exist
	_, err := s.userRepo.GetByID(ctx, *req.FromUserID)
	if err != nil {
		return nil, fmt.Errorf("source user not found: %w", err)
	}

	_, err = s.userRepo.GetByID(ctx, *req.ToUserID)
	if err != nil {
		return nil, fmt.Errorf("target user not found: %w", err)
	}

	// Check balance before creating transaction
	balance, err := s.balanceRepo.GetByUserID(ctx, *req.FromUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	if !balance.HasSufficientBalance(req.Amount) {
		return nil, fmt.Errorf("insufficient balance: have %.2f, need %.2f", balance.GetAmount(), req.Amount)
	}

	// Create transaction
	transaction, err := domain.NewTransaction(
		req.FromUserID,
		req.ToUserID,
		req.Amount,
		domain.TransactionTypeTransfer,
		req.Description,
		req.ReferenceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	// Save transaction
	if err := s.transactionRepo.Create(ctx, transaction); err != nil {
		return nil, fmt.Errorf("failed to save transaction: %w", err)
	}

	// Submit to worker pool for processing
	job := worker.NewTransactionJob(transaction.ID, &repository.Repositories{
		Transaction: s.transactionRepo,
		Balance:     s.balanceRepo,
		User:        s.userRepo,
		AuditLog:    s.auditRepo,
		Cache:       s.cacheRepo,
	})

	if err := s.workerPool.SubmitJob(job); err != nil {
		log.Error().Err(err).Str("transaction_id", transaction.ID.String()).Msg("Failed to submit transaction job")
		// Mark transaction as failed
		transaction.MarkFailed()
		s.transactionRepo.Update(ctx, transaction)
		return nil, fmt.Errorf("failed to process transaction: %w", err)
	}

	// Create audit log
	auditDetails := domain.TransactionAuditDetails{
		FromUserID:  req.FromUserID,
		ToUserID:    req.ToUserID,
		Amount:      req.Amount,
		Type:        string(domain.TransactionTypeTransfer),
		Status:      string(transaction.Status),
		Description: req.Description,
		ReferenceID: req.ReferenceID,
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeTransaction,
		domain.ActionCreate,
		transaction.ID,
		auditDetails,
		userID,
		ipAddress,
		userAgent,
	)

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("transaction_id", transaction.ID.String()).
		Float64("amount", req.Amount).
		Msg("Transfer transaction created")

	return transaction, nil
}

// GetTransaction retrieves a transaction by ID
func (s *TransactionService) GetTransaction(ctx context.Context, transactionID uuid.UUID) (*domain.Transaction, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("transaction:%s", transactionID.String())
	var cachedTransaction domain.Transaction
	if err := s.cacheRepo.Get(ctx, cacheKey, &cachedTransaction); err == nil {
		return &cachedTransaction, nil
	}

	// Get from database
	transaction, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return nil, err
	}

	// Cache for future requests if completed
	if transaction.IsCompleted() {
		s.cacheRepo.Set(ctx, cacheKey, transaction, 3600) // 1 hour
	}

	return transaction, nil
}

// GetTransactionHistory retrieves transaction history with filters
func (s *TransactionService) GetTransactionHistory(ctx context.Context, filter domain.TransactionFilter) ([]*domain.Transaction, error) {
	return s.transactionRepo.List(ctx, filter)
}

// GetUserTransactions retrieves transactions for a specific user
func (s *TransactionService) GetUserTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Transaction, error) {
	return s.transactionRepo.GetByUserID(ctx, userID, limit, offset)
}

// GetTransactionByReference retrieves a transaction by reference ID
func (s *TransactionService) GetTransactionByReference(ctx context.Context, referenceID string) (*domain.Transaction, error) {
	return s.transactionRepo.GetByReferenceID(ctx, referenceID)
}

// CancelTransaction cancels a pending transaction
func (s *TransactionService) CancelTransaction(ctx context.Context, transactionID uuid.UUID, userID *uuid.UUID, ipAddress net.IP, userAgent string) error {
	transaction, err := s.transactionRepo.GetByID(ctx, transactionID)
	if err != nil {
		return err
	}

	if !transaction.IsPending() {
		return fmt.Errorf("transaction cannot be cancelled, current status: %s", transaction.Status)
	}

	// Mark as cancelled
	transaction.MarkCancelled()
	if err := s.transactionRepo.Update(ctx, transaction); err != nil {
		return fmt.Errorf("failed to cancel transaction: %w", err)
	}

	// Create audit log
	auditDetails := domain.TransactionAuditDetails{
		FromUserID:  transaction.FromUserID,
		ToUserID:    transaction.ToUserID,
		Amount:      transaction.Amount,
		Type:        string(transaction.Type),
		Status:      string(transaction.Status),
		OldStatus:   string(domain.TransactionStatusPending),
		Description: transaction.Description,
		ReferenceID: transaction.ReferenceID,
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeTransaction,
		domain.ActionUpdate,
		transaction.ID,
		auditDetails,
		userID,
		ipAddress,
		userAgent,
	)

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("transaction_id", transaction.ID.String()).
		Msg("Transaction cancelled")

	return nil
}

// ProcessPendingTransactions processes pending transactions (for batch processing)
func (s *TransactionService) ProcessPendingTransactions(ctx context.Context, limit int) error {
	transactions, err := s.transactionRepo.ListPending(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to list pending transactions: %w", err)
	}

	for _, transaction := range transactions {
		job := worker.NewTransactionJob(transaction.ID, &repository.Repositories{
			Transaction: s.transactionRepo,
			Balance:     s.balanceRepo,
			User:        s.userRepo,
			AuditLog:    s.auditRepo,
			Cache:       s.cacheRepo,
		})

		if err := s.workerPool.SubmitJob(job); err != nil {
			log.Error().Err(err).Str("transaction_id", transaction.ID.String()).Msg("Failed to submit transaction job")
		}
	}

	log.Info().Int("count", len(transactions)).Msg("Submitted pending transactions for processing")
	return nil
}
