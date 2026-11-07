// Package handler = endpoint HTTP budget.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"financial-tracker/internal/budget/domain"
	"financial-tracker/internal/shared/owner"
	"financial-tracker/platform/httpserver"
)

// Usecase = kebutuhan handler terhadap usecase budget.
type Usecase interface {
	List(ctx context.Context, ownerID string) ([]domain.Budget, error)
	Create(ctx context.Context, ownerID string, p domain.Params) (domain.Budget, error)
	Update(ctx context.Context, ownerID, id string, period domain.Period, limit int64) (domain.Budget, error)
	Delete(ctx context.Context, ownerID, id string) error
}

type Handler struct{ uc Usecase }

func New(uc Usecase) *Handler { return &Handler{uc: uc} }

// Routes mendaftarkan route relatif terhadap /api.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/budgets", h.list)
	r.Post("/budgets", h.create)
	r.Patch("/budgets/{id}", h.update)
	r.Delete("/budgets/{id}", h.delete)
}

// ---- DTO (bentuk JSON sama dengan IBudget di josh-vault) ----

type createRequest struct {
	Category string `json:"category"`
	Period   string `json:"period"`
	Limit    int64  `json:"limit"`
}

type updateRequest struct {
	Period string `json:"period"`
	Limit  int64  `json:"limit"`
}

type budgetDTO struct {
	ID        string    `json:"_id"`
	Category  string    `json:"category"`
	Period    string    `json:"period"`
	Limit     int64     `json:"limit"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(b domain.Budget) budgetDTO {
	return budgetDTO{
		ID:        b.ID,
		Category:  b.Category,
		Period:    string(b.Period),
		Limit:     b.Limit,
		CreatedAt: b.CreatedAt.UTC(),
	}
}

// ---- handlers ----

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.uc.List(r.Context(), owner.From(r.Context()))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	out := make([]budgetDTO, 0, len(list))
	for _, b := range list {
		out = append(out, toDTO(b))
	}
	httpserver.OK(w, http.StatusOK, "Budgets fetched", out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	b, err := h.uc.Create(r.Context(), owner.From(r.Context()), domain.Params{
		Category: req.Category,
		Period:   domain.Period(req.Period),
		Limit:    req.Limit,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusCreated, "Budget created", toDTO(b))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	b, err := h.uc.Update(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id"),
		domain.Period(req.Period), req.Limit)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusOK, "Budget updated", toDTO(b))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Delete(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id")); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusOK, "Budget deleted", nil)
}
