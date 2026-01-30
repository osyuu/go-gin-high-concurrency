package model

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID          int       `json:"id" db:"id"`
	EventID     uuid.UUID `json:"event_id" db:"event_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type UpdateEventParams struct {
	Name        *string
	Description *string
}
