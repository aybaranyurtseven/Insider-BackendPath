package main

import (
	"context"
	"encoding/json"
	"fmt"
	"insider-backend/internal/config"
	"insider-backend/internal/domain"
	"insider-backend/internal/middleware"
	"insider-backend/pkg/logger"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// MockUserService simulates user operations without database
type MockUserService struct {
	users map[string]*domain.User
	jwtSecret string
	accessTTL time.Duration
}

func NewMockUserService(jwtSecret string, accessTTL time.Duration) *MockUserService {
	return &MockUserService{
		users: make(map[string]*domain.User),
		jwtSecret: jwtSecret,
		accessTTL: accessTTL,
	}
}

func (s *MockUserService) Register(ctx context.Context, req domain.CreateUserRequest) (*domain.AuthResponse, error) {
	// Check if user exists
	for _, user := range s.users {
		if user.Username == req.Username || user.Email == req.Email {
			return nil, fmt.Errorf("user already exists")
		}
	}

	// Create new user
	user, err := domain.NewUser(req.Username, req.Email, req.Password, domain.RoleUser)
	if err != nil {
		return nil, err
	}

	// Store user
	s.users[user.ID.String()] = user

	// Generate mock token (simplified)
	token := fmt.Sprintf("mock-token-%s", user.ID.String())

	return &domain.AuthResponse{
		User:         user,
		AccessToken:  token,
		RefreshToken: fmt.Sprintf("refresh-%s", token),
	}, nil
}

func (s *MockUserService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthResponse, error) {
	// Find user by username
	var foundUser *domain.User
	for _, user := range s.users {
		if user.Username == req.Username {
			foundUser = user
			break
		}
	}

	if foundUser == nil {
		return nil, fmt.Errorf("user not found")
	}

	if !foundUser.CheckPassword(req.Password) {
		return nil, fmt.Errorf("invalid password")
	}

	// Generate mock token
	token := fmt.Sprintf("mock-token-%s", foundUser.ID.String())

	return &domain.AuthResponse{
		User:         foundUser,
		AccessToken:  token,
		RefreshToken: fmt.Sprintf("refresh-%s", token),
	}, nil
}

func (s *MockUserService) ValidateToken(token string) (*MockJWTClaims, error) {
	// Simple token validation for demo
	if len(token) < 10 || token[:10] != "mock-token" {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract user ID from token (simplified)
	userIDStr := token[11:] // Skip "mock-token-"
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token format")
	}

	user, exists := s.users[userID.String()]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return &MockJWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     string(user.Role),
	}, nil
}

type MockJWTClaims struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Role     string    `json:"role"`
}

// HTTP Handlers
func registerHandler(userService *MockUserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		authResponse, err := userService.Register(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(authResponse)
	}
}

func loginHandler(userService *MockUserService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req domain.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		authResponse, err := userService.Login(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(authResponse)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0-mock",
		"mode":      "development",
		"message":   "Mock server running - database services not connected",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

func mockBalanceHandler(w http.ResponseWriter, r *http.Request) {
	balance := map[string]interface{}{
		"user_id":         "demo-user-id",
		"amount":          123.45,
		"last_updated_at": time.Now().UTC(),
		"version":         1,
		"note":           "Mock balance - no database connected",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(balance)
}

func main() {
	fmt.Println("🚀 Starting Insider Backend Mock Server...")

	// Initialize logger
	logger.Init(logger.LoggerConfig{
		Level:  "debug",
		Format: "console",
	})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Create mock user service
	userService := NewMockUserService(cfg.JWT.SecretKey, cfg.JWT.AccessTokenTTL)

	// Create router
	router := mux.NewRouter()

	// Global middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logging())
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.JSONContentType())

	// Public routes
	api.HandleFunc("/auth/register", registerHandler(userService)).Methods("POST")
	api.HandleFunc("/auth/login", loginHandler(userService)).Methods("POST")
	api.HandleFunc("/health", healthHandler).Methods("GET")

	// Mock protected routes (simplified auth)
	api.HandleFunc("/balances/current", mockBalanceHandler).Methods("GET")

	// Demo info endpoint
	api.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"message": "Insider Backend Demo API",
			"features": []string{
				"User Registration & Login",
				"JWT Authentication (mock)",
				"Health Monitoring",
				"Request Logging",
				"CORS Support",
				"Security Headers",
			},
			"endpoints": map[string]string{
				"Register": "POST /api/v1/auth/register",
				"Login":    "POST /api/v1/auth/login",
				"Health":   "GET /api/v1/health",
				"Balance":  "GET /api/v1/balances/current",
				"Demo":     "GET /api/v1/demo",
			},
			"note": "This is a demonstration server. For full functionality, start with Docker.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	}).Methods("GET")

	// Create server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	log.Info().
		Str("host", cfg.Server.Host).
		Str("port", cfg.Server.Port).
		Msg("🌟 Mock server starting")

	fmt.Printf("\n🌐 Server running at: http://localhost:%s\n", cfg.Server.Port)
	fmt.Println("\n📋 Available Endpoints:")
	fmt.Println("   • GET  /api/v1/demo          - API information")
	fmt.Println("   • GET  /api/v1/health        - Health check")
	fmt.Println("   • POST /api/v1/auth/register - Register user")
	fmt.Println("   • POST /api/v1/auth/login    - Login user")
	fmt.Println("   • GET  /api/v1/balances/current - Get balance (mock)")
	fmt.Println("\n🔗 Try: curl http://localhost:8080/api/v1/demo")
	fmt.Println("\n⏹️  Press Ctrl+C to stop")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
