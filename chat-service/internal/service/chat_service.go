package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meetohin/web-chat/chat-service/internal/client"
	"github.com/meetohin/web-chat/chat-service/internal/repository"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: Configure proper CORS for production
	},
}

type Client struct {
	conn               *websocket.Conn
	username           string
	userID             string
	send               chan []byte
	notificationCancel context.CancelFunc // for canceling notification subscription
}

type ChatService struct {
	authClient         *client.AuthClient
	messageRepo        repository.MessageRepository
	notificationClient *NotificationClient
	clients            map[*Client]bool
	broadcast          chan []byte
	register           chan *Client
	unregister         chan *Client
	mu                 sync.RWMutex
}

func NewChatService(authClient *client.AuthClient, messageRepo repository.MessageRepository, redisURL string) (*ChatService, error) {
	notificationClient, err := NewNotificationClient(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification client: %w", err)
	}

	return &ChatService{
		authClient:         authClient,
		messageRepo:        messageRepo,
		notificationClient: notificationClient,
		clients:            make(map[*Client]bool),
		broadcast:          make(chan []byte),
		register:           make(chan *Client),
		unregister:         make(chan *Client),
	}, nil
}

func (cs *ChatService) Run() {
	for {
		select {
		case client := <-cs.register:
			cs.mu.Lock()
			cs.clients[client] = true
			cs.mu.Unlock()

			cs.startNotificationSubscription(client)

			log.Printf("Client %s connected. Total clients: %d", client.username, len(cs.clients))

		case client := <-cs.unregister:
			cs.mu.Lock()
			if _, ok := cs.clients[client]; ok {
				delete(cs.clients, client)
				close(client.send)

				if client.notificationCancel != nil {
					client.notificationCancel()
				}
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
					if client.notificationCancel != nil {
						client.notificationCancel()
					}
				}
			}
			cs.mu.RUnlock()
		}
	}
}

func (cs *ChatService) startNotificationSubscription(client *Client) {
	ctx, cancel := context.WithCancel(context.Background())
	client.notificationCancel = cancel

	go cs.notificationClient.SubscribeToNotifications(ctx, client.userID, func(data []byte) {
		select {
		case client.send <- data:
		default:
			cs.unregister <- client
		}
	})
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
		userID:   username,
		send:     make(chan []byte, 256),
	}

	cs.register <- client

	// Send recent messages to newly connected client
	recentMessages, err := cs.messageRepo.GetRecentMessages(50)
	if err != nil {
		log.Printf("Error getting recent messages: %v", err)
	} else {
		for _, msg := range recentMessages {
			msgJSON, _ := json.Marshal(msg)
			select {
			case client.send <- msgJSON:
			default:
				close(client.send)
				return
			}
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

		if len(text) > 1000 {
			text = text[:1000]
		}

		message, err := cs.messageRepo.SaveMessage(client.username, text)
		if err != nil {
			log.Printf("Error saving message: %v", err)
			continue
		}

		messageJSON, _ := json.Marshal(message)
		cs.broadcast <- messageJSON

		go cs.sendNotificationToOthers(client.username, text)
	}
}

func (cs *ChatService) sendNotificationToOthers(senderUsername, messageText string) {
	cs.mu.RLock()
	var recipients []string
	for client := range cs.clients {
		if client.username != senderUsername {
			recipients = append(recipients, client.userID)
		}
	}
	cs.mu.RUnlock()

	for _, userID := range recipients {
		notification := NotificationRequest{
			UserID:  userID,
			Title:   fmt.Sprintf("New message from %s", senderUsername),
			Message: cs.truncateMessage(messageText, 100),
			Type:    "message",
		}

		go func(notif NotificationRequest) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := cs.notificationClient.SendNotification(ctx, notif); err != nil {
				log.Printf("Failed to send notification: %v", err)
			}
		}(notification)
	}
}

func (cs *ChatService) truncateMessage(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength-3] + "..."
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

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}
		}
	}
}

func (cs *ChatService) GetStats() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	messageCount, err := cs.messageRepo.GetMessageCount()
	if err != nil {
		log.Printf("Error getting message count: %v", err)
		messageCount = 0
	}

	return map[string]interface{}{
		"connected_clients": len(cs.clients),
		"total_messages":    messageCount,
	}
}

func (cs *ChatService) Close() error {
	return cs.notificationClient.Close()
}
