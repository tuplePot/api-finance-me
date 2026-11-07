// Package handler = endpoint HTTP saving challenge.
package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"financial-tracker/internal/challenge/domain"
	"financial-tracker/internal/shared/apperror"
	"financial-tracker/internal/shared/clock"
	"financial-tracker/internal/shared/owner"
	"financial-tracker/platform/httpserver"
)

// Usecase = kebutuhan handler terhadap usecase challenge.
type Usecase interface {
	List(ctx context.Context, ownerID string) ([]domain.Challenge, error)
	Create(ctx context.Context, ownerID string, p domain.Params) (domain.Challenge, error)
	Delete(ctx context.Context, ownerID, id string) error
	Log(ctx context.Context, ownerID, id string, kind domain.LogKind) (domain.Challenge, error)
}

type Handler struct{ uc Usecase }

func New(uc Usecase) *Handler { return &Handler{uc: uc} }

// Routes mendaftarkan route relatif terhadap /api.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/challenges", h.list)
	r.Post("/challenges", h.create)
	r.Delete("/challenges/{id}", h.delete)
	r.Post("/challenges/{id}/logs", h.log)
}

// ---- DTO (bentuk JSON sama dengan IChallenge di josh-vault) ----

type itemDTO struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

type createRequest struct {
	Title        string  `json:"title"`
	Temptation   itemDTO `json:"temptation"`
	Goal         itemDTO `json:"goal"`
	TargetAmount int64   `json:"targetAmount"`
	Deadline     string  `json:"deadline"`
}

type logRequest struct {
	Kind string `json:"kind"`
}

type logDTO struct {
	ID     string `json:"_id"`
	Kind   string `json:"kind"`
	Amount int64  `json:"amount"`
	Date   string `json:"date"`
}

type challengeDTO struct {
	ID           string     `json:"_id"`
	Title        string     `json:"title"`
	Temptation   itemDTO    `json:"temptation"`
	Goal         itemDTO    `json:"goal"`
	TargetAmount int64      `json:"targetAmount"`
	Deadline     string     `json:"deadline"`
	SavedAmount  int64      `json:"savedAmount"`
	Logs         []logDTO   `json:"logs"`
	Status       string     `json:"status"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

func toDTO(c domain.Challenge) challengeDTO {
	logs := make([]logDTO, 0, len(c.Logs))
	for _, l := range c.Logs {
		logs = append(logs, logDTO{
			ID: l.ID, Kind: string(l.Kind), Amount: l.Amount, Date: l.Date.Format(clock.DateLayout),
		})
	}
	return challengeDTO{
		ID:           c.ID,
		Title:        c.Title,
		Temptation:   itemDTO(c.Temptation),
		Goal:         itemDTO(c.Goal),
		TargetAmount: c.TargetAmount,
		Deadline:     c.Deadline.Format(clock.DateLayout),
		SavedAmount:  c.SavedAmount,
		Logs:         logs,
		Status:       string(c.Status),
		CompletedAt:  utcPtr(c.CompletedAt),
		CreatedAt:    c.CreatedAt.UTC(),
	}
}

// ---- handlers ----

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.uc.List(r.Context(), owner.From(r.Context()))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	out := make([]challengeDTO, 0, len(list))
	for _, c := range list {
		out = append(out, toDTO(c))
	}
	httpserver.OK(w, http.StatusOK, "Challenges fetched", out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	deadline, err := clock.ParseDate(req.Deadline)
	if err != nil {
		httpserver.WriteError(w, apperror.Invalid(errors.New("Deadline must be yyyy-mm-dd"),
			map[string]string{"deadline": "must be yyyy-mm-dd"}))
		return
	}
	c, err := h.uc.Create(r.Context(), owner.From(r.Context()), domain.Params{
		Title:        req.Title,
		Temptation:   domain.Item(req.Temptation),
		Goal:         domain.Item(req.Goal),
		TargetAmount: req.TargetAmount,
		Deadline:     deadline,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusCreated, "Challenge created", toDTO(c))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Delete(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id")); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusOK, "Challenge deleted", nil)
}

func (h *Handler) log(w http.ResponseWriter, r *http.Request) {
	var req logRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	c, err := h.uc.Log(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id"), domain.LogKind(req.Kind))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	msg := "Logged"
	if c.Status == domain.Success {
		msg = "Challenge completed!"
	}
	httpserver.OK(w, http.StatusCreated, msg, toDTO(c))
}

// utcPtr menormalkan waktu ke UTC (pgx membaca timestamptz dalam zona lokal).
func utcPtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	u := t.UTC()
	return &u
}
