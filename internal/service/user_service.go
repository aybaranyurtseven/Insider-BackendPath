package service

import (
	"context"
	"fmt"
	"insider-backend/internal/domain"
	"insider-backend/internal/repository"
	"net"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type UserService struct {
	userRepo    repository.UserRepository
	balanceRepo repository.BalanceRepository
	auditRepo   repository.AuditLogRepository
	cacheRepo   repository.CacheRepository
	jwtSecret   string
	accessTTL   time.Duration
	refreshTTL  time.Duration
}

type JWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

func NewUserService(repos *repository.Repositories, jwtSecret string, accessTTL, refreshTTL time.Duration) *UserService {
	return &UserService{
		userRepo:    repos.User,
		balanceRepo: repos.Balance,
		auditRepo:   repos.AuditLog,
		cacheRepo:   repos.Cache,
		jwtSecret:   jwtSecret,
		accessTTL:   accessTTL,
		refreshTTL:  refreshTTL,
	}
}

// Register creates a new user account
func (s *UserService) Register(ctx context.Context, req domain.CreateUserRequest, ipAddress net.IP, userAgent string) (*domain.AuthResponse, error) {
	log.Info().
		Str("username", req.Username).
		Str("email", req.Email).
		Msg("User registration attempt")

	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("username, email, and password are required")
	}

	// Set default role if not provided
	role := domain.RoleUser
	if req.Role != "" {
		role = domain.UserRole(req.Role)
	}

	// Check if user already exists
	exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("username already exists")
	}

	exists, err = s.userRepo.ExistsByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("email already exists")
	}

	// Create user
	user, err := domain.NewUser(req.Username, req.Email, req.Password, role)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Save user to database
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	// Create initial balance
	balance := domain.NewBalance(user.ID)
	if err := s.balanceRepo.Create(ctx, balance); err != nil {
		log.Error().Err(err).Str("user_id", user.ID.String()).Msg("Failed to create initial balance")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Create audit log
	auditDetails := domain.UserAuditDetails{
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeUser,
		domain.ActionCreate,
		user.ID,
		auditDetails,
		&user.ID,
		ipAddress,
		userAgent,
	)

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("user_id", user.ID.String()).
		Str("username", user.Username).
		Msg("User registered successfully")

	return &domain.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// Login authenticates a user and returns tokens
func (s *UserService) Login(ctx context.Context, req domain.LoginRequest, ipAddress net.IP, userAgent string) (*domain.AuthResponse, error) {
	log.Info().
		Str("username", req.Username).
		Msg("User login attempt")

	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		log.Warn().
			Str("username", req.Username).
			Msg("Login attempt with non-existent username")
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check password
	if !user.CheckPassword(req.Password) {
		log.Warn().
			Str("user_id", user.ID.String()).
			Str("username", req.Username).
			Msg("Login attempt with incorrect password")
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	accessToken, err := s.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.generateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Cache user session
	s.cacheUserSession(ctx, user.ID.String(), accessToken)

	// Create audit log
	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeUser,
		domain.ActionLogin,
		user.ID,
		nil,
		&user.ID,
		ipAddress,
		userAgent,
	)

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("user_id", user.ID.String()).
		Str("username", user.Username).
		Msg("User logged in successfully")

	return &domain.AuthResponse{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// GetUser retrieves a user by ID
func (s *UserService) GetUser(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	var cachedUser domain.User
	if err := s.cacheRepo.Get(ctx, cacheKey, &cachedUser); err == nil {
		return &cachedUser, nil
	}

	// Get from database
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache for future requests
	s.cacheRepo.Set(ctx, cacheKey, user, 300) // 5 minutes

	return user, nil
}

// UpdateUser updates user information
func (s *UserService) UpdateUser(ctx context.Context, userID uuid.UUID, req domain.UpdateUserRequest, ipAddress net.IP, userAgent string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	oldUser := *user // Copy for audit

	// Update fields
	if req.Username != "" {
		// Check if username is already taken
		exists, err := s.userRepo.ExistsByUsername(ctx, req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username existence: %w", err)
		}
		if exists && req.Username != user.Username {
			return nil, fmt.Errorf("username already exists")
		}
		user.Username = req.Username
	}

	if req.Email != "" {
		// Check if email is already taken
		exists, err := s.userRepo.ExistsByEmail(ctx, req.Email)
		if err != nil {
			return nil, fmt.Errorf("failed to check email existence: %w", err)
		}
		if exists && req.Email != user.Email {
			return nil, fmt.Errorf("email already exists")
		}
		user.Email = req.Email
	}

	if req.Role != "" {
		user.Role = domain.UserRole(req.Role)
	}

	user.UpdatedAt = time.Now()

	// Validate updated user
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Save to database
	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	s.cacheRepo.Delete(ctx, cacheKey)

	// Create audit log
	auditDetails := domain.UserAuditDetails{
		Username:    user.Username,
		Email:       user.Email,
		Role:        string(user.Role),
		OldUsername: oldUser.Username,
		OldEmail:    oldUser.Email,
		OldRole:     string(oldUser.Role),
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeUser,
		domain.ActionUpdate,
		user.ID,
		auditDetails,
		&user.ID,
		ipAddress,
		userAgent,
	)

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("user_id", user.ID.String()).
		Msg("User updated successfully")

	return user, nil
}

// ListUsers returns a paginated list of users
func (s *UserService) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	return s.userRepo.List(ctx, limit, offset)
}

// DeleteUser deletes a user
func (s *UserService) DeleteUser(ctx context.Context, userID uuid.UUID, ipAddress net.IP, userAgent string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := s.userRepo.Delete(ctx, userID); err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%s", userID.String())
	s.cacheRepo.Delete(ctx, cacheKey)

	// Create audit log
	auditDetails := domain.UserAuditDetails{
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
	}

	auditLog, _ := domain.NewAuditLog(
		domain.EntityTypeUser,
		domain.ActionDelete,
		user.ID,
		auditDetails,
		&user.ID,
		ipAddress,
		userAgent,
	)

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		log.Warn().Err(err).Msg("Failed to create audit log")
	}

	log.Info().
		Str("user_id", user.ID.String()).
		Msg("User deleted successfully")

	return nil
}

// ValidateToken validates a JWT token and returns claims
func (s *UserService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// generateAccessToken generates an access token for the user
func (s *UserService) generateAccessToken(user *domain.User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// generateRefreshToken generates a refresh token for the user
func (s *UserService) generateRefreshToken(user *domain.User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// cacheUserSession caches user session information
func (s *UserService) cacheUserSession(ctx context.Context, userID, token string) {
	cacheKey := fmt.Sprintf("session:%s", userID)
	s.cacheRepo.Set(ctx, cacheKey, token, int(s.accessTTL.Seconds()))
}
