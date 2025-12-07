package config

import (
	"fmt"
	"os"
)

type Config struct {
	MongoDBURI      string
	MongoDBDatabase string
	ServerPort      string
}

func LoadConfig() *Config {
	config := &Config{
		MongoDBURI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDBDatabase: getEnv("MONGODB_DATABASE", "taxihub"),
		ServerPort:      getEnv("SERVER_PORT", "9000"),
	}

	if config.MongoDBURI == "" {
		panic("MONGODB_URI is required")
	}
	if config.MongoDBDatabase == "" {
		panic("MONGODB_DATABASE is required")
	}
	if config.ServerPort == "" {
		panic("SERVER_PORT is required")
	}

	return config
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func (c *Config) GetServerAddress() string {
	return fmt.Sprintf(":%s", c.ServerPort)
}
