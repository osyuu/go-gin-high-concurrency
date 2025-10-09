package model

import "time"

// OrderStatus 訂單狀態類型
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// IsValid 驗證狀態是否有效
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusConfirmed, OrderStatusCancelled:
		return true
	}
	return false
}

// CanTransitionTo 檢查是否可以轉換到目標狀態
func (s OrderStatus) CanTransitionTo(target OrderStatus) bool {
	transitions := map[OrderStatus][]OrderStatus{
		OrderStatusPending:   {OrderStatusConfirmed, OrderStatusCancelled},
		OrderStatusConfirmed: {OrderStatusCancelled},
		OrderStatusCancelled: {}, // 不能轉換到任何狀態
	}

	allowed, ok := transitions[s]
	if !ok {
		return false
	}

	for _, status := range allowed {
		if status == target {
			return true
		}
	}
	return false
}

// Order 訂單模型
type Order struct {
	ID         int         `json:"id" db:"id"`
	UserID     int         `json:"user_id" db:"user_id"`
	TicketID   int         `json:"ticket_id" db:"ticket_id"`
	Quantity   int         `json:"quantity" db:"quantity"`
	TotalPrice float64     `json:"total_price" db:"total_price"`
	Status     OrderStatus `json:"status" db:"status"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at" db:"updated_at"`
	DeletedAt  *time.Time  `json:"deleted_at,omitempty" db:"deleted_at"`
}

// IsDeleted 檢查訂單是否已刪除
func (o *Order) IsDeleted() bool {
	return o.DeletedAt != nil
}

// CreateOrderRequest 創建訂單請求
type CreateOrderRequest struct {
	UserID   int `json:"user_id" binding:"required"`
	TicketID int `json:"ticket_id" binding:"required"`
	Quantity int `json:"quantity" binding:"required,min=1"`
}

// OrderResponse 訂單響應
type OrderResponse struct {
	ID         int     `json:"id"`
	UserID     int     `json:"user_id"`
	TicketID   int     `json:"ticket_id"`
	Quantity   int     `json:"quantity"`
	TotalPrice float64 `json:"total_price"`
	Status     string  `json:"status"`
	CreatedAt  string  `json:"created_at"`
}
