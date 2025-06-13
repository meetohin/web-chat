package main

import (
	"github.com/meetohin/web-chat/chat-service/internal/client"
	"github.com/meetohin/web-chat/chat-service/internal/database"
	"github.com/meetohin/web-chat/chat-service/internal/handler"
	"github.com/meetohin/web-chat/chat-service/internal/service"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	if authServiceURL == "" {
		authServiceURL = "auth-service:50051"
	}
	// Init client of authorization
	authClient, err := client.NewAuthClient(authServiceURL)
	if err != nil {
		log.Fatalf("Failed to connect to auth service: %v", err)
	}
	defer authClient.Close()

	// Init repo and service
	messageRepo := database.NewMessageRepository()
	chatService := service.NewChatService(authClient, messageRepo)
	chatHandler := handler.NewChatHandler(authClient, chatService)

	// Run chat service
	go chatService.Run()

	// Configure routers
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

	// Waiting for stopping signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down chat service...")
	log.Println("Chat service stopped")
}
