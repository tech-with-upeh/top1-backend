package tasks

import (
	"context"
	"time"

	"github.com/hibiken/asynq"
)

type Client struct {
	client *asynq.Client
}

func NewClient(redisURL string) (*Client, error) {
	opt, err := asynq.ParseRedisURI(redisURL)
	if err != nil {
		return nil, err
	}

	return &Client{
		client: asynq.NewClient(opt),
	}, nil
}

func (c *Client) SendEmail(
	ctx context.Context,
	to, subject, body string,
	isHTML bool,
) (*asynq.TaskInfo, error) {
	task, err := NewEmailTask(to, subject, body, isHTML)
	if err != nil {
		return nil, err
	}

	taskInfo, err := c.client.EnqueueContext(
		ctx,
		task,
		asynq.Queue(QueueEmails),
		asynq.MaxRetry(10),
		asynq.Timeout(30*time.Second),
	)

	return taskInfo, err
}

func (c *Client) Close() error {
	return c.client.Close()
}
