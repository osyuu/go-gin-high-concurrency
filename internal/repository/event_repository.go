package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-gin-high-concurrency/internal/model"
	apperrors "go-gin-high-concurrency/pkg/app_errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository interface {
	Create(ctx context.Context, event *model.Event) (*model.Event, error)
	List(ctx context.Context) ([]*model.Event, error)
	FindByID(ctx context.Context, id int) (*model.Event, error)
	FindByEventID(ctx context.Context, eventID uuid.UUID) (*model.Event, error)
	Update(ctx context.Context, id int, params model.UpdateEventParams) (*model.Event, error)
}

type EventRepositoryImpl struct {
	pool *pgxpool.Pool
}

func NewEventRepository(pool *pgxpool.Pool) EventRepository {
	return &EventRepositoryImpl{
		pool: pool,
	}
}

func (r *EventRepositoryImpl) Create(ctx context.Context, event *model.Event) (*model.Event, error) {
	query := `
		INSERT INTO events (event_id, name, description)
		VALUES ($1, $2, $3)
		RETURNING id, event_id, name, description, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		event.EventID, event.Name, event.Description,
	).Scan(
		&event.ID,
		&event.EventID,
		&event.Name,
		&event.Description,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *EventRepositoryImpl) List(ctx context.Context) ([]*model.Event, error) {
	query := `
		SELECT id, event_id, name, description, created_at, updated_at
		FROM events
		ORDER BY created_at DESC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*model.Event, 0)
	for rows.Next() {
		var event model.Event
		err := rows.Scan(
			&event.ID,
			&event.EventID,
			&event.Name,
			&event.Description,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}
	return events, nil
}

func (r *EventRepositoryImpl) FindByID(ctx context.Context, id int) (*model.Event, error) {
	query := `
		SELECT id, event_id, name, description, created_at, updated_at
		FROM events
		WHERE id = $1
	`

	var event model.Event
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&event.ID,
		&event.EventID,
		&event.Name,
		&event.Description,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrEventNotFound
		}
		return nil, err
	}

	return &event, nil
}

func (r *EventRepositoryImpl) FindByEventID(ctx context.Context, eventID uuid.UUID) (*model.Event, error) {
	query := `
		SELECT id, event_id, name, description, created_at, updated_at
		FROM events
		WHERE event_id = $1
	`

	var event model.Event
	err := r.pool.QueryRow(ctx, query, eventID).Scan(
		&event.ID,
		&event.EventID,
		&event.Name,
		&event.Description,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrEventNotFound
		}
		return nil, err
	}

	return &event, nil
}

func (r *EventRepositoryImpl) Update(ctx context.Context, id int, params model.UpdateEventParams) (*model.Event, error) {
	sets := []string{}
	args := []interface{}{}
	argPos := 1

	if params.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *params.Name)
		argPos++
	}

	if params.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argPos))
		args = append(args, *params.Description)
		argPos++
	}

	if len(sets) == 0 {
		return nil, apperrors.ErrInvalidInput
	}

	// add updated_at
	sets = append(sets, fmt.Sprintf("updated_at = $%d", argPos))
	args = append(args, time.Now().UTC())
	argPos++

	// add id
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE events
		SET %s
		WHERE id = $%d
        RETURNING id, event_id, name, description, created_at, updated_at
	`, strings.Join(sets, ", "), argPos)

	var event model.Event

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&event.ID,
		&event.EventID,
		&event.Name,
		&event.Description,
		&event.CreatedAt,
		&event.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrEventNotFound
		}
		return nil, err
	}

	return &event, nil
}
