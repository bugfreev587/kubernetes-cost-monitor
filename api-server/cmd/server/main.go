package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"

	"github.com/bugfreev587/k8s-cost-api-server/internal/api"
	"github.com/bugfreev587/k8s-cost-api-server/internal/config"
	"github.com/bugfreev587/k8s-cost-api-server/internal/db"
	"github.com/bugfreev587/k8s-cost-api-server/internal/services"
	// New import for interfaces
)

func main() {
	_ = godotenv.Load() // load .env file if exists

	environment := os.Getenv("ENVIRONMENT")
	log.Printf(" environment: %s", environment)

	confFile := ""
	if environment == "production" || environment == "prod" {
		confFile = "/app/conf/api-server-prod.yaml" // Default for production (Docker)
		// Fallback to relative path for local development
		if _, err := os.Stat(confFile); os.IsNotExist(err) {
			confFile = "./conf/api-server-prod.yaml"
		}
	} else {
		confFile = "/app/conf/api-server-dev.yaml" // Default for development (Docker)
		// Fallback to relative path for local development
		if _, err := os.Stat(confFile); os.IsNotExist(err) {
			confFile = "./conf/api-server-dev.yaml"
		}
	}
	log.Printf("--- confFile: %s", confFile)

	log.Printf("Loading config from: %s", confFile)
	cfg, err := config.LoadConfigFromPath(confFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("--- cfg: %+v", cfg)

	// postgres - check for Railway service-prefixed variables first, then DATABASE_URL, then config
	postgresDSN := cfg.Postgres.DSN
	if postgresURL := os.Getenv("POSTGRES_DB_URL"); postgresURL != "" {
		postgresDSN = postgresURL
		log.Println("Using POSTGRES_DB_URL from environment")
	}
	postgresDB, err := db.InitPostgres(postgresDSN)
	if err != nil {
		log.Fatalf("postgres init err: %v", err)
	}
	log.Println("✓ PostgreSQL Database connected")

	// timescale - check for Railway service-prefixed variables first, then alternatives, then config
	timescaleDSN := cfg.Timescale.DSN
	if timescaleDBURL := os.Getenv("TIMESCALE_DB_URL"); timescaleDBURL != "" {
		timescaleDSN = timescaleDBURL
		log.Println("Using TIMESCALE_DB_URL from environment")
	}
	timescaleDB, err := db.InitTimescale(timescaleDSN)
	if err != nil {
		log.Fatalf("timescale init err: %v", err)
	}
	log.Println("✓ Timescale Database connected")

	// redis - use REDIS_URL from Railway if available, otherwise use config
	var rdb *redis.Client
	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		// Parse REDIS_URL format: redis://[:password@]host[:port][/db]
		redisOpts, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatalf("Failed to parse REDIS_URL: %v", err)
		}
		rdb = redis.NewClient(redisOpts)
		log.Println("✓ Redis client created from REDIS_URL")
	} else {
		// Fallback to config file values
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		log.Println("✓ Redis client created from config")
	}
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping err: %v", err)
	}
	log.Println("✓ Redis connected")

	// Wrap concrete DB types and Redis client to satisfy interfaces
	postgresService := &db.PostgresServiceWrapper{PostgresDB: postgresDB}
	timescaleService := &db.TimescaleServiceWrapper{TimescaleDB: timescaleDB}
	redisService := &services.RedisClientWrapper{Client: rdb}

	// services
	apiKeySvc := services.NewAPIKeyService(postgresDB.GetPostgresDB(), []byte(cfg.Security.APIKeyPepper), rdb, time.Duration(cfg.Security.APIKeyCacheTTLSeconds)*time.Second)

	apiServer := api.NewServer(cfg, postgresService, timescaleService, redisService, apiKeySvc)
	go func() {
		if err := apiServer.Run(); err != nil {
			log.Fatalf("server start err: %v", err)
		}
	}()
	log.Printf("✓ Server started on %s:%s", cfg.Server.Host, cfg.Server.Port)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
