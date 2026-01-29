package queue_test

import (
	"context"
	"testing"
	"time"

	"go-gin-high-concurrency/internal/model"
	"go-gin-high-concurrency/internal/queue"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cleanupStream(ctx context.Context, t *testing.T) {
	t.Helper()
	_ = testRdb.Del(ctx, queue.StreamKey).Err()
}

// --- 1. 建構 ---

func TestNewRedisStreamOrderQueue(t *testing.T) {
	ctx := context.Background()
	cleanupStream(ctx, t)

	t.Run("success", func(t *testing.T) {
		q, err := queue.NewRedisStreamOrderQueue(testRdb, "test-consumer", nil)
		require.NoError(t, err)
		require.NotNil(t, q)
	})

	t.Run("empty_consumer_id_generates_uuid", func(t *testing.T) {
		cleanupStream(ctx, t)
		q, err := queue.NewRedisStreamOrderQueue(testRdb, "", nil)
		require.NoError(t, err)
		require.NotNil(t, q)
	})
}

// --- 2. 發送（基本成功即可；完整「有收到」由訂閱測試涵蓋）---

func TestRedisStreamOrderQueue_PublishOrder(t *testing.T) {
	ctx := context.Background()
	cleanupStream(ctx, t)

	q, err := queue.NewRedisStreamOrderQueue(testRdb, "pub-test", nil)
	require.NoError(t, err)

	order := &model.Order{
		UserID:     1,
		TicketID:   2,
		RequestID:  "req-1",
		Quantity:   3,
		TotalPrice: 99.0,
		Status:     model.OrderStatusPending,
	}
	err = q.PublishOrder(ctx, order)
	require.NoError(t, err)
}

// --- 3. 訂閱與投遞：驗證「發出去的內容」與「收進來的內容」一致 ---

func TestRedisStreamOrderQueue_Subscribe_deliversPublishedMessage(t *testing.T) {
	ctx := context.Background()
	cleanupStream(ctx, t)

	q, err := queue.NewRedisStreamOrderQueue(testRdb, "deliver-test", nil)
	require.NoError(t, err)

	order := &model.Order{
		UserID:     10,
		TicketID:   20,
		RequestID:  "req-deliver",
		Quantity:   1,
		TotalPrice: 50.0,
		Status:     model.OrderStatusPending,
	}
	err = q.PublishOrder(ctx, order)
	require.NoError(t, err)

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	delCh, err := q.SubscribeOrders(subCtx)
	require.NoError(t, err)

	select {
	case d, ok := <-delCh:
		require.True(t, ok, "應收到一筆")
		require.NotNil(t, d.Data)
		assert.Equal(t, order.UserID, d.Data.UserID)
		assert.Equal(t, order.TicketID, d.Data.TicketID)
		assert.Equal(t, order.RequestID, d.Data.RequestID)
		assert.Equal(t, order.Quantity, d.Data.Quantity)
		assert.Equal(t, order.TotalPrice, d.Data.TotalPrice)
		assert.Equal(t, order.Status, d.Data.Status)
	case <-subCtx.Done():
		t.Fatal("timeout 未收到訊息")
	}
}

// --- 4. Ack 結果：Ack 後該訊息不應再被投遞 ---

func TestRedisStreamOrderQueue_Ack_preventsRedelivery(t *testing.T) {
	ctx := context.Background()
	cleanupStream(ctx, t)

	q, err := queue.NewRedisStreamOrderQueue(testRdb, "ack-test", nil)
	require.NoError(t, err)

	order := &model.Order{
		UserID: 11, TicketID: 21, RequestID: "req-ack",
		Quantity: 1, TotalPrice: 60.0, Status: model.OrderStatusPending,
	}
	require.NoError(t, q.PublishOrder(ctx, order))

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	delCh, err := q.SubscribeOrders(subCtx)
	require.NoError(t, err)

	var first *model.Order
	select {
	case d, ok := <-delCh:
		require.True(t, ok)
		require.NotNil(t, d.Data)
		first = d.Data
		d.Ack()
	case <-subCtx.Done():
		t.Fatal("timeout 未收到第一筆")
	}

	// 驗證結果：下一讀應為 channel 關閉（cancel 後），不應再收到同一筆
	cancel()
	next, ok := <-delCh
	assert.False(t, ok, "Ack 後不應再投遞；下一讀應為 channel 關閉")
	if ok && next.Data != nil && next.Data.RequestID == first.RequestID {
		t.Fatalf("Ack 後不應再收到同一筆: RequestID=%s", first.RequestID)
	}
}

// --- 5. Nack(false) 結果：丟棄後該訊息不應再被投遞 ---

func TestRedisStreamOrderQueue_NackDiscard_preventsRedelivery(t *testing.T) {
	ctx := context.Background()
	cleanupStream(ctx, t)

	q, err := queue.NewRedisStreamOrderQueue(testRdb, "nack-discard-test", nil)
	require.NoError(t, err)

	order := &model.Order{
		UserID: 7, TicketID: 8, RequestID: "req-nack-discard",
		Quantity: 2, TotalPrice: 200.0, Status: model.OrderStatusPending,
	}
	require.NoError(t, q.PublishOrder(ctx, order))

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	delCh, err := q.SubscribeOrders(subCtx)
	require.NoError(t, err)

	select {
	case d, ok := <-delCh:
		require.True(t, ok)
		require.NotNil(t, d.Data)
		assert.Equal(t, order.RequestID, d.Data.RequestID)
		d.Nack(false)
	case <-subCtx.Done():
		t.Fatal("timeout 未收到第一筆")
	}

	// 驗證結果：短時間內不應再收到同一筆（已丟棄）
	select {
	case d, ok := <-delCh:
		if ok && d.Data != nil && d.Data.RequestID == order.RequestID {
			t.Fatalf("Nack(false) 後不應再投遞同一筆，表示未正確丟棄: RequestID=%s", d.Data.RequestID)
		}
	case <-time.After(2 * time.Second):
		// 2 秒內無第二次投遞，視為已丟棄
	}
	cancel()
}

// --- 6. Nack(true) 結果：重試時應在約 ClaimMinIdleTime 後再次投遞 ---

func TestRedisStreamOrderQueue_NackRequeue_redeliversAfterIdle(t *testing.T) {
	ctx := context.Background()
	cleanupStream(ctx, t)

	cfg := &queue.RedisStreamOrderQueueConfig{
		ClaimMinIdleTime:   200 * time.Millisecond,
		ReadGroupBlockTime: 500 * time.Millisecond,
	}
	q, err := queue.NewRedisStreamOrderQueue(testRdb, "nack-requeue-test", cfg)
	require.NoError(t, err)

	order := &model.Order{
		UserID: 9, TicketID: 10, RequestID: "req-requeue",
		Quantity: 1, TotalPrice: 100.0, Status: model.OrderStatusPending,
	}
	require.NoError(t, q.PublishOrder(ctx, order))

	subCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	delCh, err := q.SubscribeOrders(subCtx)
	require.NoError(t, err)

	select {
	case d, ok := <-delCh:
		require.True(t, ok)
		require.NotNil(t, d.Data)
		assert.Equal(t, order.RequestID, d.Data.RequestID)
		d.Nack(true)
	case <-subCtx.Done():
		t.Fatal("timeout 未收到第一筆")
	}

	// 驗證結果：約 5 秒後應再次收到同一筆（XAUTOCLAIM 領回）
	select {
	case d, ok := <-delCh:
		require.True(t, ok, "Nack(requeue) 後應在 ClaimMinIdleTime 後再次投遞")
		require.NotNil(t, d.Data)
		assert.Equal(t, order.RequestID, d.Data.RequestID, "重試應為同一筆")
	case <-subCtx.Done():
		t.Fatal("timeout 未收到重試投遞")
	}
}

// --- 7. 毒藥消息：超過 MaxRetryCount 後應被丟棄，不再投遞 ---

// 毒藥測試：注入短逾時與較小 MaxRetryCount，數秒內完成。
func TestRedisStreamOrderQueue_poisonMessage_discardedAfterMaxRetries(t *testing.T) {
	ctx := context.Background()
	cleanupStream(ctx, t)

	// 注入短逾時與較小重試次數，測試可在數秒內完成
	cfg := &queue.RedisStreamOrderQueueConfig{
		ClaimMinIdleTime:   200 * time.Millisecond,
		MaxRetryCount:      3,
		ReadGroupBlockTime: 200 * time.Millisecond,
	}
	q, err := queue.NewRedisStreamOrderQueue(testRdb, "poison-test", cfg)
	require.NoError(t, err)

	order := &model.Order{
		UserID: 99, TicketID: 100, RequestID: "req-poison",
		Quantity: 1, TotalPrice: 1.0, Status: model.OrderStatusPending,
	}
	require.NoError(t, q.PublishOrder(ctx, order))

	subCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	delCh, err := q.SubscribeOrders(subCtx)
	require.NoError(t, err)

	// 每次收到都 Nack(requeue)；超過 MaxRetryCount 後實作會丟棄，不再投遞
	received := 0
	waitNoMore := 500 * time.Millisecond
loop:
	for {
		select {
		case d, ok := <-delCh:
			if !ok {
				t.Fatalf("channel 提早關閉，只收到 %d 次", received)
			}
			require.NotNil(t, d.Data)
			assert.Equal(t, order.RequestID, d.Data.RequestID)
			received++
			d.Nack(true)
		case <-time.After(waitNoMore):
			if received >= 1 {
				break loop
			}
			t.Fatalf("timeout 未收到任何一筆")
		case <-subCtx.Done():
			t.Fatalf("test context timeout，只收到 %d 次", received)
		}
	}

	require.GreaterOrEqual(t, received, 1, "應至少收到 1 次")
	// 驗證結果：已不再投遞；若再收到同一筆則失敗
	select {
	case d, ok := <-delCh:
		if ok && d.Data != nil && d.Data.RequestID == order.RequestID {
			t.Fatalf("超過 MaxRetryCount 後應丟棄毒藥消息，不應再投遞: RequestID=%s", d.Data.RequestID)
		}
	case <-time.After(500 * time.Millisecond):
		// 短時間內無再投遞，視為已丟棄
	}
}

// --- 關閉行為：context 取消時 channel 關閉 ---

func TestRedisStreamOrderQueue_Subscribe_ctxCancel_closesChannel(t *testing.T) {
	ctx := context.Background()
	cleanupStream(ctx, t)

	q, err := queue.NewRedisStreamOrderQueue(testRdb, "cancel-test", nil)
	require.NoError(t, err)

	subCtx, cancel := context.WithCancel(ctx)
	delCh, err := q.SubscribeOrders(subCtx)
	require.NoError(t, err)

	cancel()
	select {
	case _, ok := <-delCh:
		assert.False(t, ok, "context 取消後 channel 應關閉")
	case <-time.After(3 * time.Second):
		t.Fatal("channel 未在時限內關閉")
	}
}
