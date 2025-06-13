package handler

import (
	"context"
	"encoding/json"
	"github.com/meetohin/web-chat/chat-service/internal/client"
	"github.com/meetohin/web-chat/chat-service/internal/service"
	"html/template"
	"net/http"
)

type ChatHandler struct {
	authClient  *client.AuthClient
	chatService *service.ChatService
	templates   *template.Template
}

func NewChatHandler(authClient *client.AuthClient, chatService *service.ChatService) *ChatHandler {
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))

	return &ChatHandler{
		authClient:  authClient,
		chatService: chatService,
		templates:   tmpl,
	}
}

func (h *ChatHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.templates.ExecuteTemplate(w, "login.html", nil)
}

func (h *ChatHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	h.templates.ExecuteTemplate(w, "register.html", nil)
}

func (h *ChatHandler) ChatPage(w http.ResponseWriter, r *http.Request) {
	h.templates.ExecuteTemplate(w, "chat.html", nil)
}

func (h *ChatHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	token, err := h.authClient.Login(context.Background(), username, password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *ChatHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	err := h.authClient.Register(context.Background(), username, password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Registration successful"})
}

func (h *ChatHandler) WebSocket(w http.ResponseWriter, r *http.Request) {
	h.chatService.HandleWebSocket(w, r)
}

func (h *ChatHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats := h.chatService.GetStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
