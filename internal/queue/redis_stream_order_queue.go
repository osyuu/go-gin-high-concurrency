package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/pkg/logger"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	StreamKey          = "orders:stream"
	ConsumerGroupName  = "order-workers"
	ConsumerNamePrefix = "worker"
)

// RedisStreamOrderQueueConfig 可注入的逾時與重試設定；nil 或零值時使用預設。
type RedisStreamOrderQueueConfig struct {
	ClaimMinIdleTime   time.Duration // PEL 中超過此時間才被 XAUTOCLAIM 領取
	MaxRetryCount      int           // 超過此次數視為毒藥消息並丟棄
	ReadGroupBlockTime time.Duration // XReadGroup 阻塞時間
}

func defaultRedisStreamConfig() RedisStreamOrderQueueConfig {
	return RedisStreamOrderQueueConfig{
		ClaimMinIdleTime:   5 * time.Second,
		MaxRetryCount:      5,
		ReadGroupBlockTime: 2 * time.Second,
	}
}

type RedisStreamOrderQueueImpl struct {
	client       *redis.Client
	streamKey    string
	groupName    string
	consumerName string
	cfg          RedisStreamOrderQueueConfig
}

// NewRedisStreamOrderQueue 建立 Redis Stream 版 OrderQueue。config 可為 nil，則使用預設逾時與重試次數。
func NewRedisStreamOrderQueue(client *redis.Client, consumerID string, config *RedisStreamOrderQueueConfig) (OrderQueue, error) {
	if consumerID == "" {
		consumerID = uuid.New().String()
	}
	cfg := defaultRedisStreamConfig()
	if config != nil {
		if config.ClaimMinIdleTime > 0 {
			cfg.ClaimMinIdleTime = config.ClaimMinIdleTime
		}
		if config.MaxRetryCount > 0 {
			cfg.MaxRetryCount = config.MaxRetryCount
		}
		if config.ReadGroupBlockTime > 0 {
			cfg.ReadGroupBlockTime = config.ReadGroupBlockTime
		}
	}
	q := &RedisStreamOrderQueueImpl{
		client:       client,
		streamKey:    StreamKey,
		groupName:    ConsumerGroupName,
		consumerName: fmt.Sprintf("%s:%s", ConsumerNamePrefix, consumerID),
		cfg:          cfg,
	}
	ctx := context.Background()
	if err := q.ensureConsumerGroup(ctx); err != nil {
		return nil, fmt.Errorf("ensure consumer group: %w", err)
	}
	return q, nil
}

func (q *RedisStreamOrderQueueImpl) ensureConsumerGroup(ctx context.Context) error {
	err := q.client.XGroupCreateMkStream(ctx, q.streamKey, q.groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (q *RedisStreamOrderQueueImpl) PublishOrder(ctx context.Context, order *model.Order) error {
	orderJSON, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("marshal order: %w", err)
	}
	_, err = q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		ID:     "*",
		Values: map[string]interface{}{"order": string(orderJSON)},
	}).Result()
	if err != nil {
		return fmt.Errorf("xadd: %w", err)
	}
	return nil
}

func (q *RedisStreamOrderQueueImpl) SubscribeOrders(ctx context.Context) (<-chan Delivery, error) {
	out := make(chan Delivery)
	go func() {
		defer close(out)
		go q.runAutoClaim(ctx, out)
		q.runReadLoop(ctx, out)
	}()
	return out, nil
}

// runReadLoop 主讀取循環：先讀 Pending("0")，再讀新消息(">")
func (q *RedisStreamOrderQueueImpl) runReadLoop(ctx context.Context, out chan<- Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			q.readAndDeliver(ctx, out)
		}
	}
}

// readAndDeliver 執行一輪讀取並投遞到 out
// 只讀 ">"（新訊息）；Pending（"0"）的訊息已由本 consumer 領過、已投遞過，不再重複投遞，改由 XAUTOCLAIM 超時後領回重試。
func (q *RedisStreamOrderQueueImpl) readAndDeliver(ctx context.Context, out chan<- Delivery) {
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.groupName,
		Consumer: q.consumerName,
		Streams:  []string{q.streamKey, ">"},
		Count:    10,
		Block:    q.cfg.ReadGroupBlockTime,
	}).Result()

	if err == redis.Nil {
		return
	}
	if err != nil {
		logger.WithComponent("mq").Error("XReadGroup failed", zap.Error(err))
		time.Sleep(time.Second)
		return
	}

	for _, stream := range streams {
		if stream.Stream != q.streamKey {
			continue
		}
		for _, msg := range stream.Messages {
			d := q.newDelivery(ctx, msg)
			if d != nil {
				select {
				case out <- *d:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

// shouldProcessMessage 檢查是否應處理（含毒藥消息判斷）
func (q *RedisStreamOrderQueueImpl) shouldProcessMessage(ctx context.Context, messageID string, isPending bool) bool {
	if !isPending {
		return true
	}
	n, err := q.getMessageRetryCount(ctx, messageID)
	if err != nil {
		logger.WithComponent("mq").Warn("getMessageRetryCount failed", zap.String("message_id", messageID), zap.Error(err))
		return true
	}
	if n >= q.cfg.MaxRetryCount {
		logger.WithComponent("mq").Warn("discard poison message", zap.String("message_id", messageID), zap.Int("retries", n), zap.Int("max_retries", q.cfg.MaxRetryCount))
		_ = q.client.XAck(ctx, q.streamKey, q.groupName, messageID).Err()
		return false
	}
	return true
}

func (q *RedisStreamOrderQueueImpl) getMessageRetryCount(ctx context.Context, messageID string) (int, error) {
	pending, err := q.client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: q.streamKey,
		Group:  q.groupName,
		Start:  messageID,
		End:    messageID,
		Count:  1,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	if len(pending) == 0 {
		return 0, nil
	}
	return int(pending[0].RetryCount), nil
}

// runAutoClaim 定時用 XAUTOCLAIM 領取超時未處理的消息
func (q *RedisStreamOrderQueueImpl) runAutoClaim(ctx context.Context, out chan<- Delivery) {
	ticker := time.NewTicker(q.cfg.ClaimMinIdleTime)
	defer ticker.Stop()
	startID := "0-0"

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			claimed, nextID, err := q.client.XAutoClaim(ctx, &redis.XAutoClaimArgs{
				Stream:   q.streamKey,
				Group:    q.groupName,
				Consumer: q.consumerName,
				MinIdle:  q.cfg.ClaimMinIdleTime,
				Count:    10,
				Start:    startID,
			}).Result()

			if err != nil && err != redis.Nil {
				logger.WithComponent("mq").Error("XAutoClaim failed", zap.Error(err))
				continue
			}
			if nextID != "" && nextID != "0-0" {
				startID = nextID
			} else {
				startID = "0-0"
			}

			for _, msg := range claimed {
				if !q.shouldProcessMessage(ctx, msg.ID, true) {
					continue
				}
				d := q.newDelivery(ctx, msg)
				if d != nil {
					select {
					case out <- *d:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}
}

// newDelivery 從 Redis 消息組裝 Delivery（含 Ack/Nack）
func (q *RedisStreamOrderQueueImpl) newDelivery(ctx context.Context, msg redis.XMessage) *Delivery {
	orderJSON, ok := msg.Values["order"].(string)
	if !ok {
		logger.WithComponent("mq").Warn("invalid message: missing order field", zap.String("message_id", msg.ID))
		return nil
	}
	var order model.Order
	if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
		logger.WithComponent("mq").Warn("unmarshal order failed", zap.String("message_id", msg.ID), zap.Error(err))
		return nil
	}
	msgID := msg.ID
	return &Delivery{
		Data: &order,
		Ack: func() {
			if err := q.client.XAck(ctx, q.streamKey, q.groupName, msgID).Err(); err != nil {
				logger.WithComponent("mq").Error("XAck failed", zap.String("message_id", msgID), zap.Error(err))
			}
		},
		Nack: func(requeue bool) {
			if requeue {
				// 不做任何事：消息留在 PEL，等 ClaimMinIdleTime 後由 XAUTOCLAIM 領取，形成延遲重試
				logger.WithComponent("mq").Info("message nack(requeue), will retry", zap.String("message_id", msgID), zap.Duration("claim_min_idle", q.cfg.ClaimMinIdleTime))
				return
			}
			if err := q.client.XAck(ctx, q.streamKey, q.groupName, msgID).Err(); err != nil {
				logger.WithComponent("mq").Error("XAck discard failed", zap.String("message_id", msgID), zap.Error(err))
			}
		},
	}
}
