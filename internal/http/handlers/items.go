package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/test-task-front/wms/internal/core/repo"
	"github.com/test-task-front/wms/internal/core/services"
)

type ItemsHandler struct {
	service *services.ItemService
	delay   time.Duration
}

func NewItemsHandler(service *services.ItemService, delay time.Duration) *ItemsHandler {
	return &ItemsHandler{
		service: service,
		delay:   delay,
	}
}

func (h *ItemsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page := parseIntWithDefault(r.URL.Query().Get("page"), 1)
	limit := parseIntWithDefault(r.URL.Query().Get("limit"), 20)
	sort := strings.TrimSpace(r.URL.Query().Get("sort"))
	dir := strings.TrimSpace(r.URL.Query().Get("dir"))

	if page < 1 {
		writeError(w, http.StatusBadRequest, "bad_request", "page must be >= 1")
		return
	}
	if limit < 1 || limit > 100 {
		writeError(w, http.StatusBadRequest, "bad_request", "limit must be between 1 and 100")
		return
	}

	filter := repo.ItemsFilter{
		Query:     q,
		Page:      page,
		Limit:     limit,
		Sort:      sort,
		Direction: dir,
	}

	result, err := h.service.List(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	if h.delay > 0 {
		time.Sleep(h.delay)
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *ItemsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}

	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *ItemsHandler) PatchQty(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid id")
		return
	}

	var req struct {
		QtyDelta *int `json:"qtyDelta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	if req.QtyDelta == nil {
		writeError(w, http.StatusBadRequest, "bad_request", "qtyDelta is required")
		return
	}

	item, err := h.service.AdjustQuantity(r.Context(), id, *req.QtyDelta)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrItemNotFound):
			writeError(w, http.StatusNotFound, "not_found", "item not found")
		case errors.Is(err, services.ErrNegativeResult):
			writeError(w, http.StatusBadRequest, "bad_request", "quantity cannot be negative")
		default:
			writeError(w, http.StatusInternalServerError, "internal", "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, item)
}

func parseIntWithDefault(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
