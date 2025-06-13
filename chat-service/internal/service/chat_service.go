package service

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/meetohin/web-chat/chat-service/internal/client"
	"github.com/meetohin/web-chat/chat-service/internal/repository"
	"log"
	"net/http"
	"sync"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // В продакшене нужна более строгая проверка
	},
}

type Client struct {
	conn     *websocket.Conn
	username string
	send     chan []byte
}

type ChatService struct {
	authClient  *client.AuthClient
	messageRepo *repository.MessageRepository
	clients     map[*Client]bool
	broadcast   chan []byte
	register    chan *Client
	unregister  chan *Client
	mu          sync.RWMutex
}

func NewChatService(authClient *client.AuthClient, messageRepo *repository.MessageRepository) *ChatService {
	return &ChatService{
		authClient:  authClient,
		messageRepo: messageRepo,
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
	}
}

func (cs *ChatService) Run() {
	for {
		select {
		case client := <-cs.register:
			cs.mu.Lock()
			cs.clients[client] = true
			cs.mu.Unlock()
			log.Printf("Client %s connected. Total clients: %d", client.username, len(cs.clients))

		case client := <-cs.unregister:
			cs.mu.Lock()
			if _, ok := cs.clients[client]; ok {
				delete(cs.clients, client)
				close(client.send)
			}
			cs.mu.Unlock()
			log.Printf("Client %s disconnected. Total clients: %d", client.username, len(cs.clients))

		case message := <-cs.broadcast:
			cs.mu.RLock()
			for client := range cs.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(cs.clients, client)
				}
			}
			cs.mu.RUnlock()
		}
	}
}

func (cs *ChatService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "Token required", http.StatusUnauthorized)
		return
	}

	username, err := cs.authClient.ValidateToken(context.Background(), token)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		conn:     conn,
		username: username,
		send:     make(chan []byte, 256),
	}

	cs.register <- client

	// Отправляем последние сообщения новому клиенту
	recentMessages := cs.messageRepo.GetRecentMessages(50)
	for _, msg := range recentMessages {
		msgJSON, _ := json.Marshal(msg)
		select {
		case client.send <- msgJSON:
		default:
			close(client.send)
			return
		}
	}

	go cs.writePump(client)
	go cs.readPump(client)
}

func (cs *ChatService) readPump(client *Client) {
	defer func() {
		cs.unregister <- client
		client.conn.Close()
	}()

	for {
		var msg map[string]interface{}
		err := client.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		text, ok := msg["text"].(string)
		if !ok || text == "" {
			continue
		}

		// Простая валидация сообщения
		if len(text) > 1000 {
			text = text[:1000]
		}

		message := cs.messageRepo.SaveMessage(client.username, text)
		messageJSON, _ := json.Marshal(message)
		cs.broadcast <- messageJSON
	}
}

func (cs *ChatService) writePump(client *Client) {
	defer client.conn.Close()

	for {
		select {
		case message, ok := <-client.send:
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			client.conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

func (cs *ChatService) GetStats() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return map[string]interface{}{
		"connected_clients": len(cs.clients),
		"total_messages":    cs.messageRepo.GetMessageCount(),
	}
}
