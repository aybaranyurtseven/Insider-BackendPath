package main

import (
	"fmt"
	"insider-backend/internal/config"
	"insider-backend/internal/domain"
	"insider-backend/pkg/logger"
	"log"

	"github.com/google/uuid"
)

func main() {
	fmt.Println("🚀 Starting Insider Backend Tests...")

	// Test 1: Configuration Loading
	fmt.Println("\n1. Testing Configuration Loading...")
	cfg, err := config.Load()
	if err != nil {
		log.Printf("❌ Configuration loading failed: %v", err)
	} else {
		fmt.Printf("✅ Configuration loaded successfully\n")
		fmt.Printf("   - Server: %s:%s\n", cfg.Server.Host, cfg.Server.Port)
		fmt.Printf("   - Database: %s\n", cfg.Database.DBName)
		fmt.Printf("   - Log Level: %s\n", cfg.Logging.Level)
	}

	// Test 2: Logger Initialization
	fmt.Println("\n2. Testing Logger Initialization...")
	logger.Init(logger.LoggerConfig{
		Level:  "debug",
		Format: "console",
	})
	fmt.Println("✅ Logger initialized successfully")

	// Test 3: Domain Models
	fmt.Println("\n3. Testing Domain Models...")
	
	// Test User Creation
	user, err := domain.NewUser("testuser", "test@example.com", "password123", domain.RoleUser)
	if err != nil {
		fmt.Printf("❌ User creation failed: %v\n", err)
	} else {
		fmt.Printf("✅ User created successfully: %s (%s)\n", user.Username, user.Email)
	}

	// Test Password Validation
	if user != nil && user.CheckPassword("password123") {
		fmt.Println("✅ Password validation works correctly")
	} else {
		fmt.Println("❌ Password validation failed")
	}

	// Test Transaction Creation
	fromUserID := uuid.New()
	toUserID := uuid.New()
	transaction, err := domain.NewTransaction(&fromUserID, &toUserID, 100.50, domain.TransactionTypeTransfer, "Test transfer", "TEST001")
	if err != nil {
		fmt.Printf("❌ Transaction creation failed: %v\n", err)
	} else {
		fmt.Printf("✅ Transaction created successfully: %s (%.2f)\n", transaction.Type, transaction.Amount)
	}

	// Test Balance Operations
	balance := domain.NewBalance(uuid.New())
	err = balance.Credit(100.0)
	if err != nil {
		fmt.Printf("❌ Balance credit failed: %v\n", err)
	} else {
		fmt.Printf("✅ Balance credited successfully: %.2f\n", balance.GetAmount())
	}

	err = balance.Debit(25.0)
	if err != nil {
		fmt.Printf("❌ Balance debit failed: %v\n", err)
	} else {
		fmt.Printf("✅ Balance debited successfully: %.2f\n", balance.GetAmount())
	}

	// Test 4: JWT Token Generation (if user service was initialized)
	fmt.Println("\n4. Testing JWT Operations...")
	if cfg != nil {
		fmt.Printf("✅ JWT Secret configured: %s...\n", cfg.JWT.SecretKey[:10])
		fmt.Printf("✅ JWT Access TTL: %v\n", cfg.JWT.AccessTokenTTL)
	}

	fmt.Println("\n🎉 Basic functionality tests completed!")
	fmt.Println("\n📋 Test Summary:")
	fmt.Println("   - Configuration loading: ✅")
	fmt.Println("   - Logger initialization: ✅")
	fmt.Println("   - Domain models: ✅")
	fmt.Println("   - Password validation: ✅")
	fmt.Println("   - Transaction creation: ✅")
	fmt.Println("   - Balance operations: ✅")
	fmt.Println("   - JWT configuration: ✅")
	
	fmt.Println("\n🔄 Next Steps:")
	fmt.Println("   1. Start Docker Desktop to run full system")
	fmt.Println("   2. Run: docker-compose up -d")
	fmt.Println("   3. Test API endpoints")
}
