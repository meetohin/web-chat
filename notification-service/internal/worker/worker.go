// internal/worker/worker.go
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/meetohin/web-chat/notification-service/internal/model"
	"github.com/meetohin/web-chat/notification-service/internal/redis"
	"github.com/meetohin/web-chat/notification-service/internal/repository"
)

type Worker struct {
	redis     *redis.Client
	repo      *repository.Repository
	logger    *log.Logger
	workers   int
	queueName string
}

func New(redis *redis.Client, repo *repository.Repository, workers int) *Worker {
	return &Worker{
		redis:     redis,
		repo:      repo,
		logger:    log.New(os.Stdout, "Worker: ", log.LstdFlags),
		workers:   workers,
		queueName: "notifications",
	}
}

func (w *Worker) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	w.logger.Printf("Starting %d workers", w.workers)

	var wg sync.WaitGroup

	for i := 0; i < w.workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.worker(ctx, workerID)
		}(i)
	}

	wg.Wait()
	w.logger.Println("All workers stopped")
	return nil
}

func (w *Worker) worker(ctx context.Context, workerID int) {
	if ctx.Err() != nil {
		return
	}

	w.logger.Printf("Worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			w.logger.Printf("Worker %d stopped", workerID)
			return
		default:
			w.processMessage(ctx, workerID)
		}
	}
}

func (w *Worker) processMessage(ctx context.Context, workerID int) {
	if ctx.Err() != nil {
		return
	}

	data, err := w.redis.Dequeue(ctx, w.queueName, 5*time.Second)
	if err != nil {
		w.logger.Printf("Worker %d: Failed to dequeue: %v", workerID, err)
		return
	}

	if data == nil {
		return
	}

	var req model.NotificationRequest
	if err := json.Unmarshal(data, &req); err != nil {
		w.logger.Printf("Worker %d: Failed to unmarshal: %v", workerID, err)
		return
	}

	w.logger.Printf("Worker %d: Processing notification for user %s", workerID, req.UserID)

	notification, err := w.repo.CreateNotification(ctx, &req)
	if err != nil {
		w.logger.Printf("Worker %d: Failed to save notification: %v", workerID, err)
		return
	}

	if err := w.sendWebSocketNotification(ctx, notification); err != nil {
		w.logger.Printf("Worker %d: Failed to send WebSocket notification: %v", workerID, err)
	}

	w.logger.Printf("Worker %d: Successfully processed notification %s", workerID, notification.ID)
}

func (w *Worker) sendWebSocketNotification(ctx context.Context, notification *model.Notification) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	channel := fmt.Sprintf("user:%s:notifications", notification.UserID)

	message := model.WebSocketMessage{
		Type: "notification",
		Data: *notification,
	}

	return w.redis.Publish(ctx, channel, message)
}

func (w *Worker) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	size, err := w.redis.QueueSize(ctx, w.queueName)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"queue_size": size,
		"workers":    w.workers,
	}, nil
}
