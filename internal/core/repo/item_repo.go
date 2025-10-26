package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/test-task-front/wms/internal/core"
)

type ItemsFilter struct {
	Query     string
	Page      int
	Limit     int
	Sort      string
	Direction string
}

type ItemRepository interface {
	List(ctx context.Context, filter ItemsFilter) (core.ItemsPage, error)
	GetByID(ctx context.Context, id string) (*core.Item, error)
	UpdateQtyByDelta(ctx context.Context, id string, delta int) (*core.Item, error)
}

type SQLItemRepository struct {
	db *sqlx.DB
}

func NewItemRepository(db *sqlx.DB) *SQLItemRepository {
	return &SQLItemRepository{db: db}
}

func (r *SQLItemRepository) List(ctx context.Context, filter ItemsFilter) (core.ItemsPage, error) {
	var page core.ItemsPage
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	sortColumn := map[string]string{
		"name":       "name",
		"sku":        "sku",
		"qty":        "qty",
		"updated_at": "updated_at",
	}[filter.Sort]
	if sortColumn == "" {
		sortColumn = "name"
	}

	sortDir := strings.ToUpper(filter.Direction)
	if sortDir != "DESC" {
		sortDir = "ASC"
	}

	whereParts := []string{"1=1"}
	args := make([]any, 0, 3)
	argIndex := 1
	if filter.Query != "" {
		whereParts = append(whereParts, fmt.Sprintf("(sku ILIKE $%d OR name ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+filter.Query+"%")
		argIndex++
	}

	whereClause := ""
	if len(whereParts) > 0 {
		whereClause = "WHERE " + strings.Join(whereParts, " AND ")
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(1) FROM items %s`, whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return page, err
	}

	offset := (filter.Page - 1) * filter.Limit
	listArgs := append([]any{}, args...)
	listArgs = append(listArgs, filter.Limit, offset)

	query := fmt.Sprintf(`
		SELECT id, sku, name, qty, location, updated_at
		FROM items
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortColumn, sortDir, argIndex, argIndex+1)

	items := make([]core.Item, 0, filter.Limit)
	if err := r.db.SelectContext(ctx, &items, query, listArgs...); err != nil {
		return page, err
	}

	page.Items = items
	page.Page = filter.Page
	page.PageSize = filter.Limit
	page.Total = total

	return page, nil
}

func (r *SQLItemRepository) GetByID(ctx context.Context, id string) (*core.Item, error) {
	var item core.Item
	if err := r.db.GetContext(ctx, &item, `
		SELECT id, sku, name, qty, location, updated_at
		FROM items
		WHERE id = $1
	`, id); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SQLItemRepository) UpdateQtyByDelta(ctx context.Context, id string, delta int) (*core.Item, error) {
	var item core.Item
	err := r.db.QueryRowxContext(ctx, `
		UPDATE items
		SET qty = qty + $2
		WHERE id = $1 AND qty + $2 >= 0
		RETURNING id, sku, name, qty, location, updated_at
	`, id, delta).StructScan(&item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
