package redis

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type Client struct {
	rdb    *redis.Client
	logger *log.Logger
}

func New(redisURL string) (*Client, error) {
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{
		rdb:    rdb,
		logger: log.New(os.Stdout, "Redis: ", log.LstdFlags),
	}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Enqueue(ctx context.Context, queue string, data interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.rdb.LPush(ctx, queue, jsonData).Err()
}

func (c *Client) Dequeue(ctx context.Context, queue string, timeout time.Duration) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	result, err := c.rdb.BRPop(ctx, timeout, queue).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Timeout
		}
		return nil, err
	}

	if len(result) < 2 {
		return nil, nil
	}

	return []byte(result[1]), nil
}

func (c *Client) Publish(ctx context.Context, channel string, data interface{}) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.rdb.Publish(ctx, channel, jsonData).Err()
}

func (c *Client) QueueSize(ctx context.Context, queue string) (int64, error) {
	if ctx.Err() != nil {
		return -1, ctx.Err()
	}

	return c.rdb.LLen(ctx, queue).Result()
}
