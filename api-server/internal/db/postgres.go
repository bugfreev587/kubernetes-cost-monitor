package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bugfreev587/k8s-cost-api-server/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	// New import for api_types
	"github.com/bugfreev587/k8s-cost-api-server/internal/app_interfaces" // Import app_interfaces
)

// Ensure PostgresDB implements app_interfaces.PostgresService. This is the key change.
var _ app_interfaces.PostgresService = (*PostgresDB)(nil)

type PostgresDB struct {
	db *gorm.DB
}

// InitPostgres creates a PostgreSQL connection with retries & pooling.
func InitPostgres(dsn string) (*PostgresDB, error) {

	// Production-ready logger (info level, slow query warnings)
	gormLogger := logger.New(
		log.New(log.Writer(), "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             500 * time.Millisecond,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	var db *gorm.DB
	var err error

	// Retry loop — wait until Postgres is ready (container startup latency)
	maxAttempts := 10
	for i := 1; i <= maxAttempts; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			PrepareStmt: true, // improves performance
			Logger:      gormLogger,
		})
		if err == nil {
			// Perform ping validation
			sqlDB, err2 := db.DB()
			if err2 == nil && sqlDB.Ping() == nil {
				fmt.Println("✓ PostgreSQL Database connected")
			}
			break
		}

		fmt.Printf("Postgres not ready (attempt %d/%d): %v\n", i, maxAttempts, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("postgres connection failed: %w", err)
	}

	// Apply connection pool settings for production readiness
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("postgres sqlDB instance error: %w", err)
	}

	sqlDB.SetMaxOpenConns(30) // maximum open connections
	sqlDB.SetMaxIdleConns(10) // idle connection pool
	sqlDB.SetConnMaxLifetime(60 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	return &PostgresDB{db: db}, nil
}

// GetPostgresDB returns the GORM DB instance.
func (p *PostgresDB) GetPostgresDB() *gorm.DB {
	return p.db
}

// CloseDB closes PostgreSQL DB connection.
func (p *PostgresDB) CloseDB() {
	if p.db == nil {
		return
	}
	sqlDB, err := p.db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func (p *PostgresDB) Health(ctx context.Context) error {
	sqlDB, err := p.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func (p *PostgresDB) GetRecommendations(c context.Context) ([]models.Recommendation, error) {
	var recs []models.Recommendation
	if err := p.db.Find(&recs).Error; err != nil {
		return nil, err
	}
	return recs, nil
}

func (p *PostgresDB) DismissRecommendation(id int64) error {
	var rec models.Recommendation
	if err := p.db.First(&rec, id).Error; err != nil {
		return err
	}
	rec.Status = "dismissed"
	return p.db.Save(&rec).Error
}

func (p *PostgresDB) ApplyRecommendation(id int64) error {
	var rec models.Recommendation
	if err := p.db.First(&rec, id).Error; err != nil {
		return err
	}
	rec.Status = "applied"
	return p.db.Save(&rec).Error
}
