// Package handler = endpoint HTTP habit.
package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"financial-tracker/internal/habit/domain"
	"financial-tracker/internal/shared/apperror"
	"financial-tracker/internal/shared/clock"
	"financial-tracker/internal/shared/owner"
	"financial-tracker/platform/httpserver"
)

// Usecase = kebutuhan handler terhadap usecase habit.
type Usecase interface {
	List(ctx context.Context, ownerID string) ([]domain.Habit, error)
	Create(ctx context.Context, ownerID string, p domain.Params) (domain.Habit, error)
	Update(ctx context.Context, ownerID, id string, patch domain.Patch) (domain.Habit, error)
	Delete(ctx context.Context, ownerID, id string) error
}

type Handler struct{ uc Usecase }

func New(uc Usecase) *Handler { return &Handler{uc: uc} }

// Routes mendaftarkan route relatif terhadap /api.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/habits", h.list)
	r.Post("/habits", h.create)
	r.Patch("/habits/{id}", h.update)
	r.Delete("/habits/{id}", h.delete)
}

// ---- DTO (bentuk JSON sama dengan IAvoidedHabit di josh-vault) ----

type createRequest struct {
	Name      string `json:"name"`
	Icon      string `json:"icon"`
	Cost      int64  `json:"cost"`
	Frequency string `json:"frequency"`
	Since     string `json:"since"`
}

// updateRequest = PATCH: field yang tidak dikirim tidak diubah.
type updateRequest struct {
	Name      *string `json:"name"`
	Icon      *string `json:"icon"`
	Cost      *int64  `json:"cost"`
	Frequency *string `json:"frequency"`
	Since     *string `json:"since"`
}

type habitDTO struct {
	ID        string    `json:"_id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Cost      int64     `json:"cost"`
	Frequency string    `json:"frequency"`
	Since     string    `json:"since"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(h domain.Habit) habitDTO {
	return habitDTO{
		ID:        h.ID,
		Name:      h.Name,
		Icon:      h.Icon,
		Cost:      h.Cost,
		Frequency: string(h.Frequency),
		Since:     h.Since.Format(clock.DateLayout),
		CreatedAt: h.CreatedAt.UTC(),
	}
}

func errBadSince() error {
	return apperror.Invalid(errors.New("Since must be yyyy-mm-dd"),
		map[string]string{"since": "must be yyyy-mm-dd"})
}

// ---- handlers ----

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.uc.List(r.Context(), owner.From(r.Context()))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	out := make([]habitDTO, 0, len(list))
	for _, x := range list {
		out = append(out, toDTO(x))
	}
	httpserver.OK(w, http.StatusOK, "Habits fetched", out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	since, err := clock.ParseDate(req.Since)
	if err != nil {
		httpserver.WriteError(w, errBadSince())
		return
	}
	x, err := h.uc.Create(r.Context(), owner.From(r.Context()), domain.Params{
		Name:      req.Name,
		Icon:      req.Icon,
		Cost:      req.Cost,
		Frequency: domain.Frequency(req.Frequency),
		Since:     since,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusCreated, "Habit created", toDTO(x))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	patch := domain.Patch{Name: req.Name, Icon: req.Icon, Cost: req.Cost}
	if req.Frequency != nil {
		f := domain.Frequency(*req.Frequency)
		patch.Frequency = &f
	}
	if req.Since != nil {
		since, err := clock.ParseDate(*req.Since)
		if err != nil {
			httpserver.WriteError(w, errBadSince())
			return
		}
		patch.Since = &since
	}
	x, err := h.uc.Update(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id"), patch)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusOK, "Habit updated", toDTO(x))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Delete(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id")); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusOK, "Habit deleted", nil)
}
