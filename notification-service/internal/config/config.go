package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL string
	RedisURL    string
	Workers     int
}

func Load() *Config {
	config := &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost/notifications?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/1"),
		Workers:     getIntEnv("WORKERS", 2),
	}

	log.Printf("Config loaded: Workers=%d", config.Workers)
	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
