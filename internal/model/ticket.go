package model

import (
	"time"

	"github.com/google/uuid"
)

// Ticket 票券模型
type Ticket struct {
	ID             int        `json:"id" db:"id"`
	TicketID       uuid.UUID  `json:"ticket_id" db:"ticket_id"`
	EventID        int        `json:"event_id" db:"event_id"`
	Name           string     `json:"name" db:"name"`
	Price          float64    `json:"price" db:"price"`
	TotalStock     int        `json:"total_stock" db:"total_stock"`
	RemainingStock int        `json:"remaining_stock" db:"remaining_stock"`
	MaxPerUser     int        `json:"max_per_user" db:"max_per_user"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`

	Event *Event `json:"event" db:"-"`
}

type UpdateTicketParams struct {
	Name       *string
	Price      *float64
	MaxPerUser *int
}

// IsDeleted 檢查票券是否已刪除
func (t *Ticket) IsDeleted() bool {
	return t.DeletedAt != nil
}

// IsAvailable 檢查票券是否可購買
func (t *Ticket) IsAvailable() bool {
	return !t.IsDeleted() && t.RemainingStock > 0
}

// TicketResponse 票券響應
type TicketResponse struct {
	ID             int     `json:"id"`
	EventID        int     `json:"event_id"`
	Name           string  `json:"name"`
	Price          float64 `json:"price"`
	TotalStock     int     `json:"total_stock"`
	RemainingStock int     `json:"remaining_stock"`
	Available      bool    `json:"available"`
}
