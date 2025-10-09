package repository

import (
	"context"
	"fmt"
	"go-gin-high-concurrency/internal/model"
	apperrors "go-gin-high-concurrency/pkg/app_errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TicketRepository interface {
	Create(ctx context.Context, ticket *model.Ticket) (*model.Ticket, error)
	List(ctx context.Context) ([]*model.Ticket, error)
	FindByID(ctx context.Context, id int) (*model.Ticket, error)
	Update(ctx context.Context, id int, ticket map[string]interface{}) (*model.Ticket, error)
	Delete(ctx context.Context, id int) error

	// Transaction methods
	FindByIDWithLock(ctx context.Context, tx pgx.Tx, id int) (*model.Ticket, error)
	IncrementStock(ctx context.Context, tx pgx.Tx, id int, quantity int) error
	DecrementStock(ctx context.Context, tx pgx.Tx, id int, quantity int) error
	AddStock(ctx context.Context, tx pgx.Tx, id int, quantity int) error
}

type TicketRepositoryImpl struct {
	pool *pgxpool.Pool
}

func NewTicketRepository(pool *pgxpool.Pool) TicketRepository {
	return &TicketRepositoryImpl{
		pool: pool,
	}
}

func (r *TicketRepositoryImpl) Create(ctx context.Context, ticket *model.Ticket) (*model.Ticket, error) {
	query := `
		INSERT INTO tickets (
		event_id, event_name, price, total_stock, remaining_stock, max_per_user)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, event_id, event_name, price, total_stock, 
			remaining_stock, max_per_user, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		ticket.EventID, ticket.EventName, ticket.Price,
		ticket.TotalStock, ticket.RemainingStock, ticket.MaxPerUser,
	).Scan(
		&ticket.ID,
		&ticket.EventID,
		&ticket.EventName,
		&ticket.Price,
		&ticket.TotalStock,
		&ticket.RemainingStock,
		&ticket.MaxPerUser,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return ticket, nil
}

func (r *TicketRepositoryImpl) List(ctx context.Context) ([]*model.Ticket, error) {
	query := `
		SELECT id, event_id, event_name, price,
				total_stock, remaining_stock, max_per_user,
				created_at, updated_at, deleted_at
		FROM tickets
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tickets := make([]*model.Ticket, 0)

	for rows.Next() {
		var ticket model.Ticket
		err := rows.Scan(
			&ticket.ID,
			&ticket.EventID,
			&ticket.EventName,
			&ticket.Price,
			&ticket.TotalStock,
			&ticket.RemainingStock,
			&ticket.MaxPerUser,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
			&ticket.DeletedAt,
		)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, &ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tickets, nil
}

func (r *TicketRepositoryImpl) FindByID(ctx context.Context, id int) (*model.Ticket, error) {
	query := `
		SELECT id, event_id, event_name, price,
				total_stock, remaining_stock, max_per_user,
				created_at, updated_at, deleted_at
		FROM tickets
		WHERE id = $1 AND deleted_at IS NULL
	`

	var ticket model.Ticket
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ticket.ID,
		&ticket.EventID,
		&ticket.EventName,
		&ticket.Price,
		&ticket.TotalStock,
		&ticket.RemainingStock,
		&ticket.MaxPerUser,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.DeletedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrTicketNotFound
		}
		return nil, err
	}

	return &ticket, nil
}

func (r *TicketRepositoryImpl) FindByIDWithLock(ctx context.Context, tx pgx.Tx, id int) (*model.Ticket, error) {
	query := `
		SELECT id, event_id, event_name, price,
				total_stock, remaining_stock, max_per_user,
				created_at, updated_at, deleted_at
		FROM tickets
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE
	`

	var ticket model.Ticket
	err := tx.QueryRow(ctx, query, id).Scan(
		&ticket.ID,
		&ticket.EventID,
		&ticket.EventName,
		&ticket.Price,
		&ticket.TotalStock,
		&ticket.RemainingStock,
		&ticket.MaxPerUser,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.DeletedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrTicketNotFound
		}
		return nil, err
	}

	return &ticket, nil
}

func (r *TicketRepositoryImpl) Update(ctx context.Context, id int, values map[string]interface{}) (*model.Ticket, error) {
	allowedFields := map[string]bool{
		"event_name":   true,
		"price":        true,
		"max_per_user": true,
	}

	sets := []string{}
	args := []interface{}{}
	argPos := 1

	for column, value := range values {
		if ok := allowedFields[column]; !ok {
			return nil, apperrors.ErrInvalidInput
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", column, argPos))
		args = append(args, value)
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
		UPDATE tickets
		SET %s
		WHERE id = $%d
        RETURNING id, event_id, event_name, price, total_stock, 
                  remaining_stock, max_per_user, created_at, updated_at
	`, strings.Join(sets, ", "), argPos)

	var ticket model.Ticket

	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&ticket.ID,
		&ticket.EventID,
		&ticket.EventName,
		&ticket.Price,
		&ticket.TotalStock,
		&ticket.RemainingStock,
		&ticket.MaxPerUser,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrTicketNotFound
		}
		return nil, err
	}

	return &ticket, nil
}

func (r *TicketRepositoryImpl) IncrementStock(ctx context.Context, tx pgx.Tx, id int, quantity int) error {
	query := `
		UPDATE tickets
		SET remaining_stock = remaining_stock + $1, updated_at = $2
		WHERE id = $3
	`

	result, err := tx.Exec(ctx, query, quantity, time.Now().UTC(), id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrTicketNotFound
	}

	return nil
}

func (r *TicketRepositoryImpl) DecrementStock(ctx context.Context, tx pgx.Tx, id int, quantity int) error {
	query := `
		UPDATE tickets
		SET remaining_stock = remaining_stock - $1, updated_at = $2
		WHERE id = $3 AND remaining_stock >= $1
	`

	result, err := tx.Exec(ctx, query, quantity, time.Now().UTC(), id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrInsufficientStock
	}

	return nil
}

func (r *TicketRepositoryImpl) Delete(ctx context.Context, id int) error {
	query := `
		UPDATE tickets
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := r.pool.Exec(ctx, query, time.Now().UTC(), time.Now().UTC(), id)
	if err != nil {
		return err
	}

	// check if ticket exists and not already deleted
	if result.RowsAffected() == 0 {
		return apperrors.ErrTicketNotFound
	}

	return nil
}

func (r *TicketRepositoryImpl) AddStock(ctx context.Context, tx pgx.Tx, id int, quantity int) error {
	if quantity <= 0 {
		return apperrors.ErrInvalidInput
	}

	query := `
		UPDATE tickets
		SET total_stock = total_stock + $1,
			remaining_stock = remaining_stock + $1,
			updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	result, err := tx.Exec(ctx, query, quantity, time.Now().UTC(), id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrTicketNotFound
	}

	return nil
}
