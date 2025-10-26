package core

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

type Item struct {
	ID        uuid.UUID `db:"id" json:"id"`
	SKU       string    `db:"sku" json:"sku"`
	Name      string    `db:"name" json:"name"`
	Qty       int       `db:"qty" json:"qty"`
	Location  string    `db:"location" json:"location"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type RefreshToken struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	Token     string     `db:"token"`
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at"`
	CreatedAt time.Time  `db:"created_at"`
}

type ItemsPage struct {
	Items    []Item `json:"items"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Total    int    `json:"total"`
}
