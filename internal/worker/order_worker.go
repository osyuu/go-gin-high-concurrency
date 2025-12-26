package worker

import (
	"context"
	"go-gin-high-concurrency/internal/queue"
	"go-gin-high-concurrency/internal/service"
)

type OrderWorker interface {
	// 訂閱訂單隊列
	Start(ctx context.Context) error
}

type OrderWorkerImpl struct {
	service service.OrderService
	queue   queue.OrderQueue
}

func NewOrderWorker(service service.OrderService, queue queue.OrderQueue) OrderWorker {
	return &OrderWorkerImpl{
		service: service,
		queue:   queue,
	}
}

func (w *OrderWorkerImpl) Start(ctx context.Context) error {
	// 1. 從自製的 MemoryQueue 訂閱
	msgs, _ := w.queue.SubscribeOrders(ctx)

	go func() {
		for msg := range msgs {
			// Worker 正在努力工作：
			// 它是那個把「訊息」變成「資料庫成果」的搬運工
			err := w.service.DispatchOrder(ctx, msg.Data)

			if err != nil {
				// 如果資料庫暫時連不上，Worker 決定重試
				msg.Nack(true)
			} else {
				// 成功了，Worker 告訴 Queue 可以結案了
				msg.Ack()
			}
		}
	}()
	return nil
}
