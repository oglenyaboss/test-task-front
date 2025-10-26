package repo

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/test-task-front/wms/internal/core"
)

type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*core.User, error)
	GetByID(ctx context.Context, id string) (*core.User, error)
}

type SQLUserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *SQLUserRepository {
	return &SQLUserRepository{db: db}
}

func (r *SQLUserRepository) GetByEmail(ctx context.Context, email string) (*core.User, error) {
	var user core.User
	if err := r.db.GetContext(ctx, &user, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`, email); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *SQLUserRepository) GetByID(ctx context.Context, id string) (*core.User, error) {
	var user core.User
	if err := r.db.GetContext(ctx, &user, `
		SELECT id, email, password_hash, created_at
		FROM users
		WHERE id = $1
	`, id); err != nil {
		return nil, err
	}
	return &user, nil
}
