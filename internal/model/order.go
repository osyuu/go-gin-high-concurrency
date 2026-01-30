package model

import (
	"time"

	"github.com/google/uuid"
)

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

// Order 訂單模型
type Order struct {
	ID         int         `json:"-" db:"id"` // 內部主鍵，不對外暴露
	OrderID    uuid.UUID   `json:"order_id" db:"order_id"`
	UserID     int         `json:"user_id" db:"user_id"`
	TicketID   int         `json:"ticket_id" db:"ticket_id"`
	RequestID  string      `json:"request_id" db:"request_id"` // 訂單請求ID, 防止重複請求
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
