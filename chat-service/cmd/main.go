package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"github.com/meetohin/web-chat/chat-service/internal/client"
	"github.com/meetohin/web-chat/chat-service/internal/handler"
	"github.com/meetohin/web-chat/chat-service/internal/repository"
	"github.com/meetohin/web-chat/chat-service/internal/service"
)

func main() {
	log.Println("Starting Chat Service...")

	// Auth client
	authServiceURL := getEnv("AUTH_SERVICE_URL", "localhost:50051")
	authClient, err := client.NewAuthClient(authServiceURL)
	if err != nil {
		log.Fatalf("Failed to connect to auth service: %v", err)
	}
	defer authClient.Close()

	// Database connection
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("DB_NAME", "webchat")
	dbUser := getEnv("DB_USER", "postgres_user")
	dbPassword := getEnv("DB_PASSWORD", "postgres")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	log.Printf("Connecting to database: host=%s port=%s dbname=%s user=%s", dbHost, dbPort, dbName, dbUser)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL database")

	// Init PostgreSQL message repository
	messageRepo := repository.NewPostgreSQLMessageRepository(db)

	// Create tables if not exist
	if postgresRepo, ok := messageRepo.(*repository.PostgreSQLMessageRepository); ok {
		// Type assertion to access CreateTables method
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

	// Chat service with notification
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/1")
	chatService, err := service.NewChatService(authClient, messageRepo, redisURL)
	if err != nil {
		log.Fatalf("Failed to create chat service: %v", err)
	}
	defer chatService.Close()

	// Handlers
	chatHandler := handler.NewChatHandler(authClient, chatService)

	// Start chat service
	go chatService.Run()

	// Routes
	http.HandleFunc("/", chatHandler.LoginPage)
	http.HandleFunc("/login", chatHandler.LoginPage)
	http.HandleFunc("/register", chatHandler.RegisterPage)
	http.HandleFunc("/chat", chatHandler.ChatPage)
	http.HandleFunc("/api/login", chatHandler.Login)
	http.HandleFunc("/api/register", chatHandler.Register)
	http.HandleFunc("/api/stats", chatHandler.Stats)
	http.HandleFunc("/ws", chatHandler.WebSocket)

	// Static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// Graceful shutdown
	serverPort := getEnv("PORT", "8080")
	server := &http.Server{Addr: ":" + serverPort}

	go func() {
		log.Printf("Chat service is running on port %s", serverPort)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down chat service...")

	// Graceful shutdown with timeout
	if err := server.Close(); err != nil {
		log.Printf("Server close error: %v", err)
	}

	log.Println("Chat service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
