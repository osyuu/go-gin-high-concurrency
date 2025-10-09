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

type UpdateUserParams struct {
	Name *string
}

type UserRepository interface {
	Create(ctx context.Context, user *model.User) (*model.User, error)
	List(ctx context.Context) ([]*model.User, error)
	FindByID(ctx context.Context, id int) (*model.User, error)
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	Update(ctx context.Context, id int, user UpdateUserParams) (*model.User, error)
	Delete(ctx context.Context, id int) error
}

type UserRepositoryImpl struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &UserRepositoryImpl{
		pool: pool,
	}
}

func (r *UserRepositoryImpl) Create(ctx context.Context, user *model.User) (*model.User, error) {
	query := `
		INSERT INTO users (name, email)
		VALUES ($1, $2)
		RETURNING id, name, email, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		user.Name, user.Email,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *UserRepositoryImpl) List(ctx context.Context) ([]*model.User, error) {
	query := `
		SELECT id, name, email, created_at, updated_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		var user model.User
		err := rows.Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, nil
}

func (r *UserRepositoryImpl) FindByID(ctx context.Context, id int) (*model.User, error) {
	query := `
		SELECT id, name, email, created_at, updated_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, name, email, created_at, updated_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	var user model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepositoryImpl) Update(ctx context.Context, id int, user UpdateUserParams) (*model.User, error) {
	sets := []string{}
	args := []interface{}{}
	argPos := 1

	if user.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *user.Name)
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
		UPDATE users
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
		RETURNING id, name, email, 
		created_at, updated_at
	`, strings.Join(sets, ", "), argPos)

	var updatedUser model.User
	err := r.pool.QueryRow(ctx, query, args...).Scan(
		&updatedUser.ID,
		&updatedUser.Name,
		&updatedUser.Email,
		&updatedUser.CreatedAt,
		&updatedUser.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, err
	}

	return &updatedUser, nil
}

func (r *UserRepositoryImpl) Delete(ctx context.Context, id int) error {
	query := `
		UPDATE users
		SET deleted_at = $1, updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL
	`

	now := time.Now().UTC()
	result, err := r.pool.Exec(ctx, query, now, now, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}
