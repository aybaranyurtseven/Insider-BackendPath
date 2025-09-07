package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"insider-backend/internal/config"
	"insider-backend/internal/handler"
	"insider-backend/internal/middleware"
	"insider-backend/internal/repository"
	"insider-backend/internal/repository/postgres"
	redisrepo "insider-backend/internal/repository/redis"
	"insider-backend/internal/service"
	"insider-backend/internal/worker"
	"insider-backend/pkg/logger"
	"insider-backend/pkg/shutdown"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type Server struct {
	config      *config.Config
	httpServer  *http.Server
	db          *sql.DB
	redisClient *redis.Client
	workerPool  *worker.WorkerPool
	router      *mux.Router
}

func New(cfg *config.Config) *Server {
	return &Server{
		config: cfg,
		router: mux.NewRouter(),
	}
}

func (s *Server) Start() error {
	// Initialize logger
	logger.Init(logger.LoggerConfig{
		Level:  s.config.Logging.Level,
		Format: s.config.Logging.Format,
	})

	log.Info().Msg("Starting server...")

	// Initialize database
	if err := s.initDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize Redis
	if err := s.initRedis(); err != nil {
		return fmt.Errorf("failed to initialize Redis: %w", err)
	}

	// Initialize worker pool
	s.initWorkerPool()

	// Setup routes
	s.setupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.config.Server.Host, s.config.Server.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	// Setup graceful shutdown
	shutdown.Init(30 * time.Second)
	shutdown.Add(s.gracefulShutdown)

	// Start server in goroutine
	go func() {
		log.Info().
			Str("host", s.config.Server.Host).
			Str("port", s.config.Server.Port).
			Msg("HTTP server starting")

		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
		}
	}()

	log.Info().Msg("Server started successfully")

	// Wait for shutdown signal
	shutdown.Wait()

	return nil
}

func (s *Server) initDatabase() error {
	log.Info().Msg("Connecting to database...")

	db, err := sql.Open("postgres", s.config.DatabaseURL())
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(s.config.Database.MaxOpen)
	db.SetMaxIdleConns(s.config.Database.MaxIdle)
	db.SetConnMaxLifetime(time.Hour)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	s.db = db
	log.Info().Msg("Database connection established")

	return nil
}

func (s *Server) initRedis() error {
	log.Info().Msg("Connecting to Redis...")

	s.redisClient = redis.NewClient(&redis.Options{
		Addr:     s.config.RedisAddr(),
		Password: s.config.Redis.Password,
		DB:       s.config.Redis.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	log.Info().Msg("Redis connection established")

	return nil
}

func (s *Server) initWorkerPool() {
	log.Info().Msg("Initializing worker pool...")

	s.workerPool = worker.NewWorkerPool(10, 1000) // 10 workers, 1000 queue size
	s.workerPool.Start()

	log.Info().Msg("Worker pool initialized")
}

func (s *Server) setupRoutes() {
	log.Info().Msg("Setting up routes...")

	// Initialize repositories
	repos := &repository.Repositories{
		User:        postgres.NewUserRepository(s.db),
		Transaction: postgres.NewTransactionRepository(s.db),
		Balance:     postgres.NewBalanceRepository(s.db),
		AuditLog:    postgres.NewAuditLogRepository(s.db),
		Cache:       redisrepo.NewCacheRepository(s.redisClient),
	}

	// Initialize services
	userService := service.NewUserService(repos, s.config.JWT.SecretKey, s.config.JWT.AccessTokenTTL, s.config.JWT.RefreshTokenTTL)
	transactionService := service.NewTransactionService(repos, s.workerPool)
	balanceService := service.NewBalanceService(repos)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	transactionHandler := handler.NewTransactionHandler(transactionService)
	balanceHandler := handler.NewBalanceHandler(balanceService)

	// Global middleware
	s.router.Use(middleware.Recovery())
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.Logging())
	s.router.Use(middleware.CORS())
	s.router.Use(middleware.SecurityHeaders())
	s.router.Use(middleware.RateLimit(100)) // 100 requests per minute
	s.router.Use(middleware.Timeout(30 * time.Second))

	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.Use(middleware.JSONContentType())
	api.Use(middleware.ValidateJSON())

	// Public routes (no authentication required)
	api.HandleFunc("/auth/register", userHandler.Register).Methods("POST")
	api.HandleFunc("/auth/login", userHandler.Login).Methods("POST")

	// Health check
	api.HandleFunc("/health", s.healthCheck).Methods("GET")

	// Protected routes (authentication required)
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware(userService))

	// User routes
	protected.HandleFunc("/users/me", userHandler.GetCurrentUser).Methods("GET")
	protected.HandleFunc("/users/{id}", userHandler.GetUser).Methods("GET")
	protected.HandleFunc("/users/{id}", userHandler.UpdateUser).Methods("PUT")

	// Admin-only user routes
	adminOnly := protected.PathPrefix("").Subrouter()
	adminOnly.Use(middleware.RoleMiddleware("admin"))
	adminOnly.HandleFunc("/users", userHandler.ListUsers).Methods("GET")
	adminOnly.HandleFunc("/users/{id}", userHandler.DeleteUser).Methods("DELETE")

	// Transaction routes
	protected.HandleFunc("/transactions/credit", transactionHandler.CreateCredit).Methods("POST")
	protected.HandleFunc("/transactions/debit", transactionHandler.CreateDebit).Methods("POST")
	protected.HandleFunc("/transactions/transfer", transactionHandler.CreateTransfer).Methods("POST")
	protected.HandleFunc("/transactions/{id}", transactionHandler.GetTransaction).Methods("GET")
	protected.HandleFunc("/transactions/{id}/cancel", transactionHandler.CancelTransaction).Methods("POST")
	protected.HandleFunc("/transactions/history", transactionHandler.GetTransactionHistory).Methods("GET")
	protected.HandleFunc("/users/{user_id}/transactions", transactionHandler.GetUserTransactions).Methods("GET")

	// Balance routes
	protected.HandleFunc("/balances/current", balanceHandler.GetCurrentBalance).Methods("GET")
	protected.HandleFunc("/balances/historical", balanceHandler.GetBalanceHistory).Methods("GET")
	protected.HandleFunc("/balances/at-time", balanceHandler.GetBalanceAtTime).Methods("GET")
	protected.HandleFunc("/balances/snapshot", balanceHandler.GetBalanceSnapshot).Methods("GET")
	protected.HandleFunc("/balances/refresh", balanceHandler.RefreshBalance).Methods("POST")
	protected.HandleFunc("/users/{user_id}/balance", balanceHandler.GetUserBalance).Methods("GET")

	log.Info().Msg("Routes configured")
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
	}

	// Check database connection
	if err := s.db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["database_error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Check Redis connection
	if err := s.redisClient.Ping(r.Context()).Err(); err != nil {
		health["status"] = "unhealthy"
		health["redis_error"] = err.Error()
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Get worker pool metrics
	metrics := s.workerPool.GetMetrics()
	health["worker_pool"] = map[string]interface{}{
		"jobs_processed":   metrics.JobsProcessed,
		"jobs_successful":  metrics.JobsSuccessful,
		"jobs_failed":      metrics.JobsFailed,
		"jobs_in_progress": metrics.JobsInProgress,
	}

	w.Header().Set("Content-Type", "application/json")
	if health["status"] == "healthy" {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(health)
}

func (s *Server) gracefulShutdown(ctx context.Context) error {
	log.Info().Msg("Starting graceful shutdown...")

	// Stop worker pool
	if s.workerPool != nil {
		log.Info().Msg("Stopping worker pool...")
		s.workerPool.Stop()
	}

	// Shutdown HTTP server
	if s.httpServer != nil {
		log.Info().Msg("Shutting down HTTP server...")
		if err := s.httpServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to shutdown HTTP server")
			return err
		}
	}

	// Close database connection
	if s.db != nil {
		log.Info().Msg("Closing database connection...")
		if err := s.db.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close database connection")
		}
	}

	// Close Redis connection
	if s.redisClient != nil {
		log.Info().Msg("Closing Redis connection...")
		if err := s.redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close Redis connection")
		}
	}

	log.Info().Msg("Graceful shutdown completed")
	return nil
}
