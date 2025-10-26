package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/test-task-front/wms/internal/core"
	"github.com/test-task-front/wms/internal/core/repo"
)

var (
	ErrItemNotFound   = errors.New("item not found")
	ErrNegativeResult = errors.New("quantity would become negative")
)

type ItemService struct {
	repo repo.ItemRepository
}

func NewItemService(repo repo.ItemRepository) *ItemService {
	return &ItemService{repo: repo}
}

func (s *ItemService) List(ctx context.Context, filter repo.ItemsFilter) (core.ItemsPage, error) {
	return s.repo.List(ctx, filter)
}

func (s *ItemService) GetByID(ctx context.Context, id string) (*core.Item, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}
	return item, nil
}

func (s *ItemService) AdjustQuantity(ctx context.Context, id string, delta int) (*core.Item, error) {
	item, err := s.repo.UpdateQtyByDelta(ctx, id, delta)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_, getErr := s.repo.GetByID(ctx, id)
			if getErr != nil {
				if errors.Is(getErr, sql.ErrNoRows) {
					return nil, ErrItemNotFound
				}
				return nil, getErr
			}
			return nil, ErrNegativeResult
		}
		return nil, err
	}
	return item, nil
}
