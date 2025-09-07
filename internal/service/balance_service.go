package service

import (
	"context"
	"fmt"
	"insider-backend/internal/domain"
	"insider-backend/internal/repository"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type BalanceService struct {
	balanceRepo repository.BalanceRepository
	userRepo    repository.UserRepository
	cacheRepo   repository.CacheRepository
}

func NewBalanceService(repos *repository.Repositories) *BalanceService {
	return &BalanceService{
		balanceRepo: repos.Balance,
		userRepo:    repos.User,
		cacheRepo:   repos.Cache,
	}
}

// GetBalance retrieves the current balance for a user
func (s *BalanceService) GetBalance(ctx context.Context, userID uuid.UUID) (*domain.Balance, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("balance:%s", userID.String())
	var cachedBalance domain.Balance
	if err := s.cacheRepo.Get(ctx, cacheKey, &cachedBalance); err == nil {
		return &cachedBalance, nil
	}

	// Get from database
	balance, err := s.balanceRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}

	// Cache for future requests
	s.cacheRepo.Set(ctx, cacheKey, balance, 60) // 1 minute

	return balance, nil
}

// GetBalanceHistory retrieves balance history for a user
func (s *BalanceService) GetBalanceHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.BalanceHistory, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	history, err := s.balanceRepo.GetHistory(ctx, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance history: %w", err)
	}

	return history, nil
}

// GetBalanceAtTime retrieves the balance at a specific point in time
func (s *BalanceService) GetBalanceAtTime(ctx context.Context, userID uuid.UUID, timestamp string) (float64, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("user not found: %w", err)
	}

	balance, err := s.balanceRepo.GetBalanceAtTime(ctx, userID, timestamp)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance at time: %w", err)
	}

	return balance, nil
}

// GetBalanceSnapshot returns a snapshot of the current balance
func (s *BalanceService) GetBalanceSnapshot(ctx context.Context, userID uuid.UUID) (domain.BalanceSnapshot, error) {
	balance, err := s.GetBalance(ctx, userID)
	if err != nil {
		return domain.BalanceSnapshot{}, err
	}

	return balance.GetSnapshot(), nil
}

// InvalidateBalanceCache invalidates the balance cache for a user
func (s *BalanceService) InvalidateBalanceCache(ctx context.Context, userID uuid.UUID) {
	cacheKey := fmt.Sprintf("balance:%s", userID.String())
	if err := s.cacheRepo.Delete(ctx, cacheKey); err != nil {
		log.Warn().Err(err).Str("user_id", userID.String()).Msg("Failed to invalidate balance cache")
	}
}

// RefreshBalance forces a refresh of the balance from the database
func (s *BalanceService) RefreshBalance(ctx context.Context, userID uuid.UUID) (*domain.Balance, error) {
	// Invalidate cache first
	s.InvalidateBalanceCache(ctx, userID)

	// Get fresh balance from database
	return s.GetBalance(ctx, userID)
}

// CreateInitialBalance creates an initial balance for a new user
func (s *BalanceService) CreateInitialBalance(ctx context.Context, userID uuid.UUID) (*domain.Balance, error) {
	// Verify user exists
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	balance := domain.NewBalance(userID)
	if err := s.balanceRepo.Create(ctx, balance); err != nil {
		return nil, fmt.Errorf("failed to create initial balance: %w", err)
	}

	log.Info().Str("user_id", userID.String()).Msg("Initial balance created")
	return balance, nil
}
