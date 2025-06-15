package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/meetohin/web-chat/notification-service/internal/model"
)

type Repository struct {
	db *sql.DB
}

func New(databaseURL string) (*Repository, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Repository{db: db}, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) CreateNotification(ctx context.Context, req *model.NotificationRequest) (*model.Notification, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	notification := &model.Notification{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		Title:     req.Title,
		Message:   req.Message,
		Type:      req.Type,
		IsRead:    false,
		CreatedAt: time.Now(),
	}

	query := `
        INSERT INTO notifications (id, user_id, title, message, type, is_read, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.db.ExecContext(ctx, query,
		notification.ID,
		notification.UserID,
		notification.Title,
		notification.Message,
		notification.Type,
		notification.IsRead,
		notification.CreatedAt,
	)

	return notification, err
}

func (r *Repository) GetUserNotifications(ctx context.Context, userID string, limit int) ([]*model.Notification, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	query := `
        SELECT id, user_id, title, message, type, is_read, created_at
        FROM notifications 
        WHERE user_id = $1 
        ORDER BY created_at DESC 
        LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*model.Notification
	for rows.Next() {
		notification := &model.Notification{}
		err := rows.Scan(
			&notification.ID,
			&notification.UserID,
			&notification.Title,
			&notification.Message,
			&notification.Type,
			&notification.IsRead,
			&notification.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

func (r *Repository) MarkAsRead(ctx context.Context, userID string, notificationIDs []string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if len(notificationIDs) == 0 {
		return nil
	}

	query := `UPDATE notifications SET is_read = TRUE WHERE user_id = $1 AND id = $2`

	for _, id := range notificationIDs {
		_, err := r.db.ExecContext(ctx, query, userID, id)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) GetUnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE`
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}
