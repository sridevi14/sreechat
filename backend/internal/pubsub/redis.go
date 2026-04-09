package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sridhar/sreechat/internal/hub"
	"github.com/sridhar/sreechat/internal/models"
)

type RedisPubSub struct {
	client     *redis.Client
	hub        *hub.Hub
	mu         sync.Mutex
	subscribed map[string]bool
}

func NewRedisPubSub(client *redis.Client, h *hub.Hub) *RedisPubSub {
	return &RedisPubSub{
		client:     client,
		hub:        h,
		subscribed: make(map[string]bool),
	}
}

func channelName(roomID string) string {
	return fmt.Sprintf("room:%s", roomID)
}

func (r *RedisPubSub) Publish(ctx context.Context, roomID string, msg *models.WSMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return r.client.Publish(ctx, channelName(roomID), data).Err()
}

// Subscribe starts a long-lived listener on a room's Redis channel.
// Uses a background context so the subscription outlives any single WS connection.
// If the goroutine dies for any reason, it cleans up the map so a future call can re-subscribe.
func (r *RedisPubSub) Subscribe(roomID string) {
	r.mu.Lock()
	if r.subscribed[roomID] {
		r.mu.Unlock()
		return
	}
	r.subscribed[roomID] = true
	r.mu.Unlock()

	sub := r.client.Subscribe(context.Background(), channelName(roomID))
	ch := sub.Channel()

	go func() {
		defer func() {
			sub.Close()
			r.mu.Lock()
			delete(r.subscribed, roomID)
			r.mu.Unlock()
			log.Printf("pubsub: unsubscribed from room %s", roomID)
		}()

		log.Printf("pubsub: subscribed to room %s", roomID)
		for redisMsg := range ch {
			var wsMsg models.WSMessage
			if err := json.Unmarshal([]byte(redisMsg.Payload), &wsMsg); err != nil {
				log.Printf("pubsub: unmarshal error: %v", err)
				continue
			}
			r.hub.BroadcastToRoom(roomID, &wsMsg)
		}
	}()
}

func (r *RedisPubSub) NextSeq(ctx context.Context, roomID string) (int64, error) {
	key := fmt.Sprintf("room:%s:seq", roomID)
	return r.client.Incr(ctx, key).Result()
}

func (r *RedisPubSub) SetOnline(ctx context.Context, userID string) error {
	key := fmt.Sprintf("online:%s", userID)
	return r.client.Set(ctx, key, "1", 35*time.Second).Err()
}

func (r *RedisPubSub) IsOnline(ctx context.Context, userID string) (bool, error) {
	key := fmt.Sprintf("online:%s", userID)
	n, err := r.client.Exists(ctx, key).Result()
	return n > 0, err
}

// ClearOnline removes the online key (e.g. user left the app).
func (r *RedisPubSub) ClearOnline(ctx context.Context, userID string) error {
	key := fmt.Sprintf("online:%s", userID)
	return r.client.Del(ctx, key).Err()
}
