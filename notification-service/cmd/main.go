package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/meetohin/web-chat/notification-service/internal/config"
	"github.com/meetohin/web-chat/notification-service/internal/redis"
	"github.com/meetohin/web-chat/notification-service/internal/repository"
	"github.com/meetohin/web-chat/notification-service/internal/worker"
)

func main() {
	log.Println("Starting Notification Service...")

	// Load config
	cfg := config.Load()

	// Connect to PostgreSQL
	repo, err := repository.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to repository: %v", err)
	}
	defer repo.Close()

	// Connect to Redis
	redisClient, err := redis.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Create and run workers
	w := worker.New(redisClient, repo, cfg.Workers)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("Shutting down...")
		cancel()
	}()

	log.Printf("Notification Service started with %d workers", cfg.Workers)

	if err := w.Start(ctx); err != nil {
		log.Printf("Worker stopped: %v", err)
	}

	log.Println("Notification Service stopped")
}
