package repo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/test-task-front/wms/internal/core"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *core.RefreshToken) error
	GetValidByToken(ctx context.Context, token string) (*core.RefreshToken, error)
	RevokeByID(ctx context.Context, id uuid.UUID) error
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
	CleanupExpired(ctx context.Context) error
}

type SQLRefreshTokenRepository struct {
	db *sqlx.DB
}

func NewRefreshTokenRepository(db *sqlx.DB) *SQLRefreshTokenRepository {
	return &SQLRefreshTokenRepository{db: db}
}

func (r *SQLRefreshTokenRepository) Create(ctx context.Context, token *core.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (id, user_id, token, expires_at, revoked_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
	`, token.ID, token.UserID, token.Token, token.ExpiresAt, token.RevokedAt)
	return err
}

func (r *SQLRefreshTokenRepository) GetValidByToken(ctx context.Context, token string) (*core.RefreshToken, error) {
	var rt core.RefreshToken
	err := r.db.GetContext(ctx, &rt, `
		SELECT id, user_id, token, expires_at, revoked_at, created_at
		FROM refresh_tokens
		WHERE token = $1
			AND (revoked_at IS NULL)
			AND expires_at > NOW()
		LIMIT 1
	`, token)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

func (r *SQLRefreshTokenRepository) RevokeByID(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *SQLRefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}

func (r *SQLRefreshTokenRepository) CleanupExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM refresh_tokens
		WHERE expires_at < $1
	`, time.Now())
	return err
}
