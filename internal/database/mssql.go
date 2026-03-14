// setup:feature:mssql

package database

import (
	"context"
	"fmt"
	"time"

	"catgoose/dothog/internal/logger"

	"github.com/catgoose/dio"
	_ "github.com/denisenkom/go-mssqldb" // Register SQL Server driver
	"github.com/jmoiron/sqlx"
)

type sqlConnectionConfig struct {
	driver   string
	server   string
	database string
	user     string
	password string
}

// getConnectionConfig gets environment variables for sqlConnectionConfig
func getConnectionConfig() (*sqlConnectionConfig, error) {
	get := dio.Env
	driver := "sqlserver"
	server, err := get("DB_HOST")
	if err != nil {
		return nil, logger.LogAndReturnError("Failed to get database server", err)
	}
	database, err := get("DB_DATABASE")
	if err != nil {
		return nil, logger.LogAndReturnError("Failed to get database name", err)
	}
	user, err := get("DB_USER")
	if err != nil {
		return nil, logger.LogAndReturnError("Failed to get database user", err)
	}
	password, err := get("DB_PASSWORD")
	if err != nil {
		return nil, logger.LogAndReturnError("Failed to get database password", err)
	}
	config := &sqlConnectionConfig{
		driver:   driver,
		server:   server,
		database: database,
		user:     user,
		password: password,
	}
	return config, nil
}

// connectionString constructs the connection string based on the config
func (config *sqlConnectionConfig) connectionString() string {
	// In development, disable encryption for easier local testing
	dev := dio.Dev()
	var encrypt string
	if dev {
		encrypt = "&encrypt=disable"
	} else {
		encrypt = ""
	}
	connection := fmt.Sprintf(
		"%s://%s:%s@%s?database=%s&secure=0%s&trustServerCertificate=1",
		config.driver,
		config.user,
		config.password,
		config.server,
		config.database,
		encrypt,
	)
	return connection
}

func openMSSQLDB(ctx context.Context) (*sqlx.DB, error) {
	log := logger.WithContext(ctx)
	config, err := getConnectionConfig()
	if err != nil {
		log.Error("Failed to load database configuration", "error", err)
		return nil, fmt.Errorf("failed to load database configuration: %w", err)
	}
	db, err := sqlx.Open(config.driver, config.connectionString())
	if err != nil {
		log.Error("Failed to connect to database", "error", err,
			"driver", config.driver, "server", config.server, "database", config.database)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		log.Error("Failed to ping database", "error", err,
			"driver", config.driver, "server", config.server, "database", config.database)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
