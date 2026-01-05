package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"
	"github.com/healthcare-market-research/backend/internal/cache"
	"github.com/healthcare-market-research/backend/internal/config"
	"github.com/healthcare-market-research/backend/internal/db"
	"github.com/healthcare-market-research/backend/internal/handler"
	"github.com/healthcare-market-research/backend/internal/middleware"
	"github.com/healthcare-market-research/backend/internal/repository"
	"github.com/healthcare-market-research/backend/internal/service"
	"github.com/healthcare-market-research/backend/pkg/logger"
	"github.com/joho/godotenv"

	_ "github.com/healthcare-market-research/backend/docs"
)

// @title Healthcare Market Research API
// @version 1.0
// @description RESTful API for healthcare market research reports and categories with caching and pagination support
// @termsOfService https://example.com/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host 192.168.1.2:8081
// @BasePath /
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @tag.name Health
// @tag.description Health check endpoints

// @tag.name Authentication
// @tag.description User authentication and token management

// @tag.name Users
// @tag.description User management operations

// @tag.name Reports
// @tag.description Operations related to healthcare market research reports

// @tag.name Categories
// @tag.description Operations related to report categories and hierarchies

// @tag.name Authors
// @tag.description Operations related to report authors and analysts
func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger.Init(cfg.Environment)
	logger.Info("Starting Healthcare Market Research API", "environment", cfg.Environment)

	// Connect to database
	if err := db.Connect(cfg); err != nil {
		logger.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		logger.Error("Failed to run migrations", "error", err)
		os.Exit(1)
	}

	// Connect to Redis (optional - app continues even if Redis fails)
	if err := cache.Connect(cfg); err != nil {
		logger.Warn("Failed to connect to Redis, caching will be disabled", "error", err)
	} else {
		logger.Info("Redis connected successfully")
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db.DB)
	categoryRepo := repository.NewCategoryRepository(db.DB)
	reportRepo := repository.NewReportRepository(db.DB)
	authorRepo := repository.NewAuthorRepository(db.DB)

	// Initialize services
	userService := service.NewUserService(userRepo)
	authService := service.NewAuthService(userRepo, &cfg.Auth)
	categoryService := service.NewCategoryService(categoryRepo)
	reportService := service.NewReportService(reportRepo)
	authorService := service.NewAuthorService(authorRepo)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	reportHandler := handler.NewReportHandler(reportService)
	authorHandler := handler.NewAuthorHandler(authorService)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Healthcare Market Research API",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	})

	// Global middleware
	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Request-ID",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Health check endpoint
	app.Get("/health", healthHandler.Check)

	// Swagger documentation
	app.Get("/swagger/*", swagger.HandlerDefault)

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Auth routes (public with rate limiting)
	auth := v1.Group("/auth")
	auth.Post("/login", middleware.RateLimit(cfg.RateLimit.LoginMaxAttempts, cfg.RateLimit.LoginWindow), authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/logout", middleware.RequireAuth(authService), authHandler.Logout)

	// User routes (requires authentication)
	users := v1.Group("/users", middleware.RequireAuth(authService))
	users.Get("/me", userHandler.GetMe)
	users.Get("/", middleware.RequireRole("admin"), userHandler.GetAll)
	users.Post("/", middleware.RequireRole("admin"), userHandler.Create)
	users.Put("/:id", middleware.RequireRole("admin"), userHandler.Update)
	users.Delete("/:id", middleware.RequireRole("admin"), userHandler.Delete)

	// Report routes (public read, protected write)
	v1.Get("/reports", reportHandler.GetAll)
	v1.Get("/reports/:slug", reportHandler.GetBySlug)
	v1.Get("/search", reportHandler.Search)
	v1.Post("/reports", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportHandler.Create)
	v1.Put("/reports/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), reportHandler.Update)
	v1.Delete("/reports/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin"), reportHandler.Delete)

	// Category routes (public read, protected write)
	v1.Get("/categories", categoryHandler.GetAll)
	v1.Get("/categories/:slug", categoryHandler.GetBySlug)
	v1.Get("/categories/:slug/reports", reportHandler.GetByCategorySlug)

	// Author routes (authenticated read, protected write)
	v1.Get("/authors", middleware.RequireAuth(authService), authorHandler.GetAll)
	v1.Get("/authors/:id", middleware.RequireAuth(authService), authorHandler.GetByID)
	v1.Post("/authors", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), authorHandler.Create)
	v1.Put("/authors/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin", "editor"), authorHandler.Update)
	v1.Delete("/authors/:id", middleware.RequireAuth(authService), middleware.RequireRole("admin"), authorHandler.Delete)

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logger.Info("Gracefully shutting down...")
		_ = app.Shutdown()
	}()

	// Start server
	port := fmt.Sprintf(":%s", cfg.Port)
	logger.Info("Server starting", "port", cfg.Port)

	if err := app.Listen(port); err != nil {
		logger.Error("Server failed to start", "error", err)
	}

	// Cleanup
	logger.Info("Running cleanup tasks...")
	if err := db.Close(); err != nil {
		logger.Error("Error closing database", "error", err)
	}
	if err := cache.Close(); err != nil {
		logger.Error("Error closing Redis", "error", err)
	}
	logger.Info("Server shutdown complete")
}
