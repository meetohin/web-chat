package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/meetohin/web-chat/chat-service/internal/client"
	"github.com/meetohin/web-chat/chat-service/internal/database"
	"github.com/meetohin/web-chat/chat-service/internal/handler"
	"github.com/meetohin/web-chat/chat-service/internal/service"
)

func main() {
	// Auth client
	authClient, err := client.NewAuthClient("localhost:50051")
	if err != nil {
		log.Fatalf("Failed to connect to auth service: %v", err)
	}
	defer authClient.Close()

	// Message repository
	messageRepo := database.NewMessageRepository()

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

	// Routs
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
	go func() {
		log.Println("Chat service is running on port 8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down chat service...")
	log.Println("Chat service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
