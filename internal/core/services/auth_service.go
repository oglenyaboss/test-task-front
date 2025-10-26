package services

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/test-task-front/wms/internal/auth"
	"github.com/test-task-front/wms/internal/core"
	"github.com/test-task-front/wms/internal/core/repo"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
)

type AuthService struct {
	users       repo.UserRepository
	refreshRepo repo.RefreshTokenRepository
	jwtManager  *auth.Manager
	refreshTTL  time.Duration
}

func NewAuthService(
	users repo.UserRepository,
	refresh repo.RefreshTokenRepository,
	jwtManager *auth.Manager,
	refreshTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:       users,
		refreshRepo: refresh,
		jwtManager:  jwtManager,
		refreshTTL:  refreshTTL,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	accessToken, err := s.jwtManager.GenerateToken(user.ID.String(), user.Email)
	if err != nil {
		return "", "", err
	}

	refreshTokenValue := uuid.NewString()
	refreshToken := &core.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     refreshTokenValue,
		ExpiresAt: time.Now().Add(s.refreshTTL),
	}

	if err := s.refreshRepo.Create(ctx, refreshToken); err != nil {
		return "", "", err
	}

	return accessToken, refreshTokenValue, nil
}

func (s *AuthService) Refresh(ctx context.Context, token string) (string, error) {
	refreshToken, err := s.refreshRepo.GetValidByToken(ctx, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidToken
		}
		return "", err
	}

	if refreshToken.ExpiresAt.Before(time.Now()) || refreshToken.RevokedAt != nil {
		return "", ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, refreshToken.UserID.String())
	if err != nil {
		return "", err
	}

	accessToken, err := s.jwtManager.GenerateToken(user.ID.String(), user.Email)
	if err != nil {
		return "", err
	}

	return accessToken, nil
}
