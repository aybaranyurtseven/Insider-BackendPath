package repository

import (
	"context"
	"insider-backend/internal/domain"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

type TransactionRepository interface {
	Create(ctx context.Context, transaction *domain.Transaction) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Transaction, error)
	Update(ctx context.Context, transaction *domain.Transaction) error
	List(ctx context.Context, filter domain.TransactionFilter) ([]*domain.Transaction, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Transaction, error)
	GetByReferenceID(ctx context.Context, referenceID string) (*domain.Transaction, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.TransactionStatus) error
	ListPending(ctx context.Context, limit int) ([]*domain.Transaction, error)
}

type BalanceRepository interface {
	Create(ctx context.Context, balance *domain.Balance) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.Balance, error)
	Update(ctx context.Context, balance *domain.Balance) error
	UpdateWithLock(ctx context.Context, balance *domain.Balance) error
	BatchUpdate(ctx context.Context, balances []*domain.Balance) error
	CreateHistory(ctx context.Context, history *domain.BalanceHistory) error
	GetHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.BalanceHistory, error)
	GetBalanceAtTime(ctx context.Context, userID uuid.UUID, timestamp string) (float64, error)
}

type AuditLogRepository interface {
	Create(ctx context.Context, auditLog *domain.AuditLog) error
	List(ctx context.Context, filter domain.AuditLogFilter) ([]*domain.AuditLog, error)
	GetByEntityID(ctx context.Context, entityType string, entityID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)
	DeleteOlderThan(ctx context.Context, days int) error
}

type CacheRepository interface {
	Set(ctx context.Context, key string, value interface{}, expiration int) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) (bool, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration int) (bool, error)
}

type Repositories struct {
	User        UserRepository
	Transaction TransactionRepository
	Balance     BalanceRepository
	AuditLog    AuditLogRepository
	Cache       CacheRepository
}
