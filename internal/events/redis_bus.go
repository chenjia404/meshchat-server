package events

import (
	"context"
	"encoding/json"
	"log/slog"

	"meshchat-server/internal/redisx"

	"github.com/redis/go-redis/v9"
)

// RedisBus publishes and consumes group-scoped events through Redis Pub/Sub.
type RedisBus struct {
	redis  *redis.Client
	logger *slog.Logger
}

func NewRedisBus(redisClient *redis.Client, logger *slog.Logger) *RedisBus {
	return &RedisBus{
		redis:  redisClient,
		logger: logger,
	}
}

func (b *RedisBus) Publish(ctx context.Context, event Envelope) error {
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	ch := redisx.GroupEventsChannel(event.GroupID)
	if event.ConversationID != "" {
		ch = redisx.DMEventsChannel(event.ConversationID)
	}
	return b.redis.Publish(ctx, ch, raw).Err()
}

func (b *RedisBus) Consume(ctx context.Context, handler func(context.Context, Envelope) error) error {
	pubsub := b.redis.PSubscribe(ctx, "chat:events:group:*", "chat:events:dm:*")
	defer func() {
		_ = pubsub.Close()
	}()

	channel := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-channel:
			if !ok {
				return nil
			}

			var event Envelope
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				b.logger.Warn("failed to decode redis event", slog.Any("error", err))
				continue
			}
			if err := handler(ctx, event); err != nil {
				b.logger.Warn("failed to handle redis event", slog.Any("error", err), slog.String("type", event.Type), slog.String("group_id", event.GroupID))
			}
		}
	}
}
