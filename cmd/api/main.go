package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"

	"github.com/webbies/otel-fiber-demo/internal/infrastructure/config"
	"github.com/webbies/otel-fiber-demo/internal/infrastructure/database"
	"github.com/webbies/otel-fiber-demo/internal/infrastructure/external"
	"github.com/webbies/otel-fiber-demo/internal/infrastructure/messaging"
	"github.com/webbies/otel-fiber-demo/internal/infrastructure/observability"
	"github.com/webbies/otel-fiber-demo/internal/middleware"
)

func main() {
	// Load environment variables
	if err := godotenv.Load("deployments/.env"); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := observability.NewLogger(cfg.Server.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Initialize telemetry
	telemetry, err := observability.NewTelemetryManager(&cfg.Telemetry)
	if err != nil {
		logger.Fatal("Failed to initialize telemetry", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetry.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown telemetry", err)
		}
	}()

	// Initialize business metrics
	metrics, err := observability.NewBusinessMetrics(telemetry.Meter())
	if err != nil {
		logger.Fatal("Failed to initialize metrics", err)
	}

	// Initialize database connections
	mongodb, err := database.NewMongoDB(&cfg.Database)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongodb.Disconnect(ctx); err != nil {
			logger.Error("Failed to disconnect from MongoDB", err)
		}
	}()

	redis, err := database.NewRedis(&cfg.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", err)
	}
	defer redis.Close()

	// Initialize Kafka
	kafkaManager := messaging.NewKafkaManager(&cfg.Kafka)

	// Initialize external clients
	mtnPayClient := external.NewMTNPayClient(&cfg.External.MTNPay)
	madapiClient := external.NewMADAPIClient(&cfg.External.MADAPI)
	soaClient := external.NewSOAClient(&cfg.External.SOA)

	// Create database indexes
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := mongodb.CreateIndexes(ctx); err != nil {
		logger.Error("Failed to create database indexes", err)
	}

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      cfg.Telemetry.ServiceName,
		ServerHeader: "Fiber",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Custom error handling with tracing
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(cors.New())

	// Add custom middleware for observability
	app.Use(middleware.RequestTracing(telemetry.Tracer()))
	app.Use(middleware.RequestMetrics(metrics))
	app.Use(middleware.RequestLogging(logger))
	app.Use(middleware.RateLimit(redis, &cfg.RateLimit))

	// Create dependencies container
	deps := &Dependencies{
		Config:       cfg,
		Logger:       logger,
		Telemetry:    telemetry,
		Metrics:      metrics,
		MongoDB:      mongodb,
		Redis:        redis,
		KafkaManager: kafkaManager,
		MTNPayClient: mtnPayClient,
		MADAPIClient: madapiClient,
		SOAClient:    soaClient,
	}

	// Setup routes
	setupRoutes(app, deps)

	// Start server
	logger.Info("Starting server on port " + cfg.Server.Port)

	// Graceful shutdown
	go func() {
		if err := app.Listen(":" + cfg.Server.Port); err != nil {
			logger.Fatal("Failed to start server", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulShutdownTimeout)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		logger.Fatal("Server forced to shutdown", err)
	}

	logger.Info("Server exited")
}

type Dependencies struct {
	Config       *config.Config
	Logger       *observability.Logger
	Telemetry    *observability.TelemetryManager
	Metrics      *observability.BusinessMetrics
	MongoDB      *database.MongoDB
	Redis        *database.Redis
	KafkaManager *messaging.KafkaManager
	MTNPayClient *external.MTNPayClient
	MADAPIClient *external.MADAPIClient
	SOAClient    *external.SOAClient
}

func setupRoutes(app *fiber.App, deps *Dependencies) {
	// Health check endpoint
	app.Get("/v1/health", healthHandler(deps))

	// API v1 group
	v1 := app.Group("/v1")

	// User endpoints
	v1.Post("/users/create", createUserHandler(deps))

	// Dashboard endpoint
	v1.Get("/dashboard", dashboardHandler(deps))

	// Payment endpoints
	v1.Post("/payments", createPaymentHandler(deps))
	v1.Get("/payments/:id/status", getPaymentStatusHandler(deps))

	// Order endpoints
	v1.Post("/orders", createOrderHandler(deps))
	v1.Get("/orders/:id", getOrderHandler(deps))

	// Reward endpoints
	v1.Post("/rewards", createRewardHandler(deps))
	v1.Get("/rewards/:userId", getUserRewardsHandler(deps))

	// Catalogue endpoint
	v1.Get("/catalogue", getCatalogueHandler(deps))

	// Unified balances endpoint
	v1.Get("/unifiedBalances/:userId", getUnifiedBalancesHandler(deps))

	// Error simulation endpoint
	v1.Post("/simulate-error", simulateErrorHandler(deps))

	// Metrics endpoint for Prometheus
	app.Get("/v1/metrics", metricsHandler(deps))
}

// Import placeholder handlers - these will be implemented next
func healthHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "healthy",
			"service":   deps.Config.Telemetry.ServiceName,
			"version":   deps.Config.Telemetry.ServiceVersion,
			"timestamp": time.Now().UTC(),
		})
	}
}

// Placeholder handlers - will be implemented with proper business logic
func createUserHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "User creation endpoint - to be implemented"})
	}
}

func dashboardHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Dashboard endpoint - to be implemented"})
	}
}

func createPaymentHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Payment creation endpoint - to be implemented"})
	}
}

func getPaymentStatusHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Payment status endpoint - to be implemented"})
	}
}

func createOrderHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Order creation endpoint - to be implemented"})
	}
}

func getOrderHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Order details endpoint - to be implemented"})
	}
}

func createRewardHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Reward creation endpoint - to be implemented"})
	}
}

func getUserRewardsHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "User rewards endpoint - to be implemented"})
	}
}

func getCatalogueHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Catalogue endpoint - to be implemented"})
	}
}

func getUnifiedBalancesHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Unified balances endpoint - to be implemented"})
	}
}

func simulateErrorHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Error simulation endpoint - to be implemented"})
	}
}

func metricsHandler(deps *Dependencies) fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "Metrics endpoint - to be implemented"})
	}
}
