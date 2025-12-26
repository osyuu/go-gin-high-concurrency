package worker

import (
	"context"
	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/queue"
	"go-gin-high-concurrency/internal/service"
	"go-gin-high-concurrency/internal/worker"
	"testing"
	"time"
)

func TestOrderWorker_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. 準備：建立自製的 Memory Queue
	q := queue.NewOrderQueue(10)

	// 2. 準備：建立一個 Mock Service 來記錄有沒有被呼叫
	// 這裡我們用一個簡單的變數或 channel 來驗證
	called := make(chan bool, 1)
	mockSvc := &mockOrderService{
		onDispatch: func(order *model.Order) {
			called <- true
		},
	}

	// 3. 啟動 Worker
	w := worker.NewOrderWorker(mockSvc, q)
	w.Start(ctx)

	// 4. 執行：模擬 API 丟入一筆訂單
	testOrder := &model.Order{ID: 1, RequestID: "TEST-123", UserID: 1, TicketID: 1, Quantity: 1, TotalPrice: 100.0, Status: model.OrderStatusPending}
	q.PublishOrder(ctx, testOrder)

	// 5. 驗證：檢查 Service 是否在時間內被觸發
	select {
	case success := <-called:
		if !success {
			t.Error("Service 被呼叫了，但結果不正確")
		}
	case <-time.After(1 * time.Second):
		t.Error("超時！Worker 沒有在時間內處理訂單")
	}
}

// 簡單的 Mock 實作
type mockOrderService struct {
	service.OrderService // 嵌入介面
	onDispatch           func(*model.Order)
}

func (m *mockOrderService) DispatchOrder(ctx context.Context, o *model.Order) error {
	m.onDispatch(o)
	return nil
}
