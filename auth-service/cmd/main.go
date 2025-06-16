package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/meetohin/web-chat/auth-service/internal/handler"
	"github.com/meetohin/web-chat/auth-service/internal/repository"
	"github.com/meetohin/web-chat/auth-service/internal/service"
	pb "github.com/meetohin/web-chat/auth-service/proto"
	"google.golang.org/grpc"
)

func main() {
	log.Println("Starting Auth Service...")

	// Database connection
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "webchat")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	log.Printf("Connecting to repository: host=%s port=%s dbname=%s user=%s", dbHost, dbPort, dbName, dbUser)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to repository: %v", err)
	}
	defer db.Close()

	// Test repository connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping repository: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL repository")

	// Init PostgreSQL repository
	userRepo := repository.NewPostgreSQLUserRepository(db)

	// Create tables if not exist
	if postgresRepo, ok := userRepo.(*repository.PostgreSQLUserRepository); ok {
		type tableCreator interface {
			CreateTables() error
		}
		if creator, ok := interface{}(postgresRepo).(tableCreator); ok {
			if err := creator.CreateTables(); err != nil {
				log.Fatalf("Failed to create tables: %v", err)
			}
			log.Println("Database tables initialized")
		}
	}

	// Init services
	authService := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authService)

	// Configure gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, authHandler)

	// Graceful shutdown
	go func() {
		log.Println("Auth service is running on port 50051")
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for termination signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down auth service...")
	s.GracefulStop()
	log.Println("Auth service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
