package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/taxihub/driver-service/internal/config"
	"github.com/taxihub/driver-service/internal/handlers"
	"github.com/taxihub/driver-service/internal/repository"
	"github.com/taxihub/driver-service/internal/service"
)

func main() {
	// Load configuration from environment
	cfg := config.LoadConfig()
	log.Printf("Configuration loaded:")
	log.Printf("  MongoDB URI: %s", cfg.MongoDBURI)
	log.Printf("  MongoDB Database: %s", cfg.MongoDBDatabase)
	log.Printf("  Server Port: %s", cfg.ServerPort)

	// Initialize database manager
	dbManager := config.NewDatabaseManager(cfg)

	// Connect to MongoDB
	log.Println("Connecting to MongoDB...")
	if err := dbManager.Initialize(); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := dbManager.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}()
	log.Println("Successfully connected to MongoDB")

	// Set up graceful shutdown for database
	dbManager.SetupGracefulShutdown()

	// Initialize dependencies
	mongoDB := dbManager.GetMongoDB()
	driverRepo := repository.NewMongoDriverRepository(mongoDB)
	driverService := service.NewDriverService(driverRepo)
	driverHandler := handlers.NewDriverHandler(driverService)

	// Initialize Fiber app with middleware
	app := fiber.New(fiber.Config{
		AppName:      "TaxiHub Driver Service",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: defaultErrorHandler,
	})

	// Add middleware
	app.Use(recover.New()) // Recover from panics
	app.Use(requestid.New()) // Add request ID for tracing
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] [${id}] ${status} - ${method} ${path} ${latency}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Health check endpoint with database status
	app.Get("/health", func(c *fiber.Ctx) error {
		// Check database health
		dbStatus := "healthy"
		if err := dbManager.HealthCheck(); err != nil {
			dbStatus = fmt.Sprintf("unhealthy: %v", err)
		}

		return c.JSON(fiber.Map{
			"status":    "ok",
			"service":   "driver-service",
			"timestamp": time.Now().UTC(),
			"database":  dbStatus,
			"version":   "1.0.0",
		})
	})

	// Register driver routes
	driverHandler.RegisterRoutes(app)

	// Log registered routes
	app.Get("/routes", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"routes": []fiber.Map{
				{
					"method": "GET",
					"path":   "/",
					"handler": "Root endpoint",
				},
				{
					"method": "GET",
					"path":   "/health",
					"handler": "Health check",
				},
				{
					"method": "GET",
					"path":   "/routes",
					"handler": "List all registered routes",
				},
				{
					"method": "POST",
					"path":   "/api/v1/drivers",
					"handler": "Create driver",
				},
				{
					"method": "GET",
					"path":   "/api/v1/drivers",
					"handler": "List drivers with pagination",
				},
				{
					"method": "GET",
					"path":   "/api/v1/drivers/:id",
					"handler": "Get driver by ID",
				},
				{
					"method": "PUT",
					"path":   "/api/v1/drivers/:id",
					"handler": "Update driver",
				},
				{
					"method": "DELETE",
					"path":   "/api/v1/drivers/:id",
					"handler": "Delete driver",
				},
				{
					"method": "GET",
					"path":   "/api/v1/drivers/nearby",
					"handler": "Find nearby drivers",
				},
				{
					"method": "PUT",
					"path":   "/api/v1/drivers/:id/location",
					"handler": "Update driver location",
				},
			},
		})
	})

	// Root endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message":  "TaxiHub Driver Service",
			"version":  "1.0.0",
			"endpoints": fiber.Map{
				"health": "/health",
				"api":    "/api/v1",
			},
		})
	})

	// Set up graceful shutdown for the server
	setupGracefulShutdown(app, cfg)

	// Startup logs
	log.Println("=== TaxiHub Driver Service ===")
	log.Printf("Server starting on %s", cfg.GetServerAddress())
	log.Printf("Health check available at http://localhost:%s/health", cfg.ServerPort)
	log.Printf("API base path: http://localhost:%s/api/v1", cfg.ServerPort)
	log.Println("Press Ctrl+C to stop the server")
	log.Println("================================")

	// Start server
	if err := app.Listen(cfg.GetServerAddress()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// defaultErrorHandler handles errors and returns JSON responses
func defaultErrorHandler(c *fiber.Ctx, err error) error {
	// Default 500 status
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	// Check if it's a Fiber error
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
		message = e.Message
	}

	// Log the error
	log.Printf("Error: %v (Status: %d, Path: %s)", err, code, c.Path())

	// Return JSON error response
	return c.Status(code).JSON(fiber.Map{
		"error": fiber.Map{
			"code":    code,
			"message": message,
			"path":    c.Path(),
			"method":  c.Method(),
		},
	})
}

// setupGracefulShutdown handles graceful server shutdown
func setupGracefulShutdown(app *fiber.App, cfg *config.Config) {
	// Create a channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal in a goroutine
	go func() {
		sig := <-sigChan
		log.Printf("\nReceived signal: %v. Shutting down gracefully...", sig)

		// Create a context with timeout for shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Shutdown the server
		if err := app.ShutdownWithContext(ctx); err != nil {
			log.Printf("Error during server shutdown: %v", err)
		}

		log.Println("Server shutdown complete")
	}()
}