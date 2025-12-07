package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type DatabaseManager struct {
	mongoDB *MongoDB
	config  *Config
}

func NewDatabaseManager(config *Config) *DatabaseManager {
	return &DatabaseManager{
		config: config,
	}
}

func (dm *DatabaseManager) Initialize() error {
	mongoDB, err := ConnectMongoDB(dm.config.MongoDBURI, dm.config.MongoDBDatabase)
	if err != nil {
		return err
	}

	dm.mongoDB = mongoDB
	return nil
}

func (dm *DatabaseManager) GetMongoDB() *MongoDB {
	return dm.mongoDB
}

func (dm *DatabaseManager) Close() error {
	if dm.mongoDB != nil {
		return dm.mongoDB.Disconnect()
	}
	return nil
}

func (dm *DatabaseManager) SetupGracefulShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v. Shutting down gracefully...", sig)

		if err := dm.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}

		os.Exit(0)
	}()
}

func (dm *DatabaseManager) HealthCheck() error {
	if dm.mongoDB == nil {
		return ErrDatabaseNotConnected
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := dm.mongoDB.PingWithContext(ctx); err != nil {
		return err
	}

	return nil
}

var (
	ErrDatabaseNotConnected = fmt.Errorf("database not connected")
)
