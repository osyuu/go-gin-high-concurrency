package queue

import (
	"context"
	"go-gin-high-concurrency/internal/model"
)

type Delivery struct {
	Data *model.Order
	Ack  func()
	Nack func(requeue bool)
}

type OrderQueue interface {
	// 發送訂單到隊列
	PublishOrder(ctx context.Context, order *model.Order) error
	// 訂閱訂單隊列
	SubscribeOrders(ctx context.Context) (<-chan Delivery, error)
}

type OrderQueueImpl struct {
	// 使用 Go channel 來模擬 MQ 隊列
	ch chan *model.Order
}

func NewOrderQueue(bufferSize int) OrderQueue {
	return &OrderQueueImpl{
		ch: make(chan *model.Order, bufferSize),
	}
}

func (q *OrderQueueImpl) PublishOrder(ctx context.Context, order *model.Order) error {
	// 模擬 MQ 發送
	q.ch <- order
	return nil
}

func (q *OrderQueueImpl) SubscribeOrders(ctx context.Context) (<-chan Delivery, error) {
	out := make(chan Delivery)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case order, ok := <-q.ch:
				if !ok {
					return
				}

				// 將原始 Order 包裝成 Delivery 格式給 Worker
				out <- Delivery{
					Data: order,
					Ack:  func() { /* 記憶體版不用做特別動作 */ },
					Nack: func(requeue bool) {
						if requeue {
							q.ch <- order // 簡單模擬重回隊列
						}
					},
				}
			}
		}
	}()

	return out, nil
}
