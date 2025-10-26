package seed

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

const (
	seedEmail    = "test@example.com"
	seedPassword = "Password123!"
)

func Run(ctx context.Context, db *sqlx.DB) error {
	if err := seedUser(ctx, db); err != nil {
		return fmt.Errorf("seed user: %w", err)
	}
	if err := seedItems(ctx, db); err != nil {
		return fmt.Errorf("seed items: %w", err)
	}
	return nil
}

func seedUser(ctx context.Context, db *sqlx.DB) error {
	var count int
	if err := db.GetContext(ctx, &count, `SELECT COUNT(1) FROM users WHERE email = $1`, seedEmail); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(seedPassword), 12)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash)
		VALUES ($1, $2, $3)
	`, uuid.New(), seedEmail, string(hash))
	return err
}

func seedItems(ctx context.Context, db *sqlx.DB) error {
	var count int
	if err := db.GetContext(ctx, &count, `SELECT COUNT(1) FROM items`); err != nil {
		return err
	}
	if count >= 120 {
		return nil
	}

	rng := rand.New(rand.NewSource(42))
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	locationIdx := 0
	locations := buildLocations()

	for i := 1; i <= 120; i++ {
		sku := fmt.Sprintf("SKU-%04d", i)
		name := fmt.Sprintf("Widget %d", i)
		qty := rng.Intn(501)
		location := locations[locationIdx%len(locations)]
		locationIdx++

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO items (id, sku, name, qty, location)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (sku) DO NOTHING
		`, uuid.New(), sku, name, qty, location); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func buildLocations() []string {
	locations := make([]string, 0, 120)
	for aisle := 1; aisle <= 10; aisle++ {
		for bin := 1; bin <= 12; bin++ {
			locations = append(locations, fmt.Sprintf("A-%02d-%02d", aisle, bin))
		}
	}
	return locations
}
