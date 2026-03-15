package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/infrasense/backend/internal/api"
	"github.com/infrasense/backend/internal/auth"
	"github.com/infrasense/backend/internal/config"
	"github.com/infrasense/backend/internal/db"
	"github.com/infrasense/backend/internal/models"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	database, err := db.NewDB(db.Config{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		Database: cfg.Database.Database,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run database migrations
	log.Println("Validating database schema and applying migrations...")
	if err := database.RunMigrations("migrations"); err != nil {
		log.Printf("FATAL: Database migration failed: %v", err)
		log.Println("Server cannot start with incompatible or failed database schema")
		os.Exit(1)
	}

	log.Println("Database schema validation and migrations completed successfully")

	// Seed default admin user if none exists
	userRepo := db.NewUserRepository(database)
	if err := seedAdminUser(context.Background(), userRepo); err != nil {
		log.Printf("Warning: Failed to seed admin user: %v", err)
	}

	// Initialize JWT service
	jwtService := auth.NewJWTService(cfg.Auth.JWTSecret)

	// Create and setup API server
	server := api.NewServer(database, jwtService, cfg)
	server.SetupRoutes()

	// Start HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      server.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // 0 = no write timeout; required for SSE streams
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting API server on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Start agent timeout detection background job
	agentTimeoutCtx, cancelAgentTimeout := context.WithCancel(context.Background())
	defer cancelAgentTimeout()

	deviceRepo := db.NewDeviceRepository(database)
	go runAgentTimeoutDetection(agentTimeoutCtx, deviceRepo)


	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Cancel background jobs
	cancelAgentTimeout()

	// Graceful shutdown with 5 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// seedAdminUser ensures at least one admin user exists in the database.
// If no users exist, it creates the default admin account.
// Credentials: admin / Admin@123456
func seedAdminUser(ctx context.Context, userRepo *db.UserRepository) error {
	// Check if any users exist
	users, err := userRepo.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %w", err)
	}

	if len(users) > 0 {
		log.Printf("Admin seed: %d user(s) already exist, skipping seed", len(users))
		return nil
	}

	log.Println("No users found — seeding default admin user...")

	// Generate bcrypt hash for default password
	passwordHash, err := auth.HashPassword("Admin@123456")
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	adminEmail := "admin@infrasense.local"
	adminUser := &models.User{
		ID:           uuid.New(),
		Username:     "admin",
		Email:        &adminEmail,
		PasswordHash: passwordHash,
		Role:         models.RoleAdmin,
		Enabled:      true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := userRepo.Create(ctx, adminUser); err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Println("==================================================")
	log.Println("  Default admin user created successfully")
	log.Println("  Username: admin")
	log.Println("  Password: Admin@123456")
	log.Println("  IMPORTANT: Change this password after first login!")
	log.Println("==================================================")
	return nil
}

// runAgentTimeoutDetection runs a background job that checks for agent timeouts every 1 minute.
// When an agent (linux_agent or windows_agent) has not sent metrics for 5 minutes,
// it marks the device as unavailable.
func runAgentTimeoutDetection(ctx context.Context, deviceRepo *db.DeviceRepository) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("Starting agent timeout detection background job")

	// Run immediately on startup
	checkAgentTimeouts(ctx, deviceRepo)

	for {
		select {
		case <-ctx.Done():
			log.Println("Agent timeout detection background job stopped")
			return
		case <-ticker.C:
			checkAgentTimeouts(ctx, deviceRepo)
		}
	}
}

// checkAgentTimeouts checks for agents that have timed out and marks them as unavailable
func checkAgentTimeouts(ctx context.Context, deviceRepo *db.DeviceRepository) {
	timeout := 5 * time.Minute
	count, err := deviceRepo.MarkTimedOutAgentsUnavailable(ctx, timeout)
	if err != nil {
		log.Printf("Error checking agent timeouts: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Marked %d agent(s) as unavailable due to timeout", count)
	}
}
