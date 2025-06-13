package main

import (
	"github.com/meetohin/web-chat/auth-service/internal/handler"
	"github.com/meetohin/web-chat/auth-service/internal/repository"
	"github.com/meetohin/web-chat/auth-service/internal/service"
	pb "github.com/meetohin/web-chat/auth-service/proto"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Init components
	userRepo := repository.NewUserRepository()
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

	// Waiting signal for stopping
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down auth service...")
	s.GracefulStop()
	log.Println("Auth service stopped")
}
