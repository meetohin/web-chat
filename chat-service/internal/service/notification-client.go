package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
)

type NotificationRequest struct {
	UserID  string `json:"user_id"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

type NotificationClient struct {
	redis  *redis.Client
	logger *log.Logger
}

func NewNotificationClient(redisURL string) (*NotificationClient, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &NotificationClient{
		redis:  rdb,
		logger: log.New(os.Stdout, "NotificationClient: ", log.LstdFlags),
	}, nil
}

func (nc *NotificationClient) Close() error {
	return nc.redis.Close()
}

func (nc *NotificationClient) SendNotification(ctx context.Context, req NotificationRequest) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if req.Type == "" {
		req.Type = "message"
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	err = nc.redis.LPush(ctx, "notifications", data).Err()
	if err != nil {
		nc.logger.Printf("Failed to send notification: %v", err)
		return err
	}

	nc.logger.Printf("Notification sent for user %s", req.UserID)
	return nil
}

func (nc *NotificationClient) SubscribeToNotifications(ctx context.Context, userID string, handler func([]byte)) {
	if ctx.Err() != nil {
		return
	}

	channel := fmt.Sprintf("user:%s:notifications", userID)

	pubsub := nc.redis.Subscribe(ctx, channel)
	defer pubsub.Close()

	nc.logger.Printf("Subscribed to notifications for user %s", userID)

	ch := pubsub.Channel()
	for {
		select {
		case msg := <-ch:
			if msg != nil {
				handler([]byte(msg.Payload))
			}
		case <-ctx.Done():
			nc.logger.Printf("Unsubscribed from notifications for user %s", userID)
			return
		}
	}
}
