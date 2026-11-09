// Package handler = endpoint HTTP transaksi.
package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"financial-tracker/internal/shared/apperror"
	"financial-tracker/internal/shared/clock"
	"financial-tracker/internal/shared/owner"
	"financial-tracker/internal/transaction/domain"
	"financial-tracker/platform/httpserver"
)

// Usecase = kebutuhan handler terhadap usecase transaksi.
type Usecase interface {
	List(ctx context.Context, ownerID string) ([]domain.Transaction, error)
	Create(ctx context.Context, ownerID string, p domain.Params) (domain.Transaction, error)
	Update(ctx context.Context, ownerID, id string, patch domain.Patch) (domain.Transaction, error)
	Delete(ctx context.Context, ownerID, id string) error
}

// Handler = handler HTTP transaksi.
type Handler struct{ uc Usecase }

func New(uc Usecase) *Handler { return &Handler{uc: uc} }

// Routes mendaftarkan route relatif terhadap /api.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/transactions", h.list)
	r.Post("/transactions", h.create)
	r.Patch("/transactions/{id}", h.update)
	r.Delete("/transactions/{id}", h.delete)
}

// ---- DTO (bentuk JSON sama dengan ITransaction di josh-vault) ----

type createRequest struct {
	Type     string `json:"type"`
	Amount   int64  `json:"amount"`
	Category string `json:"category"`
	Date     string `json:"date"`
	Note     string `json:"note"`
}

// updateRequest = PATCH: field yang tidak dikirim tidak diubah; note "" = hapus.
type updateRequest struct {
	Type     *string `json:"type"`
	Amount   *int64  `json:"amount"`
	Category *string `json:"category"`
	Date     *string `json:"date"`
	Note     *string `json:"note"`
}

type transactionDTO struct {
	ID        string    `json:"_id"`
	Type      string    `json:"type"`
	Amount    int64     `json:"amount"`
	Category  string    `json:"category"`
	Date      string    `json:"date"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(t domain.Transaction) transactionDTO {
	return transactionDTO{
		ID:        t.ID,
		Type:      string(t.Type),
		Amount:    t.Amount,
		Category:  t.Category,
		Date:      t.Date.Format(clock.DateLayout),
		Note:      t.Note,
		CreatedAt: t.CreatedAt.UTC(),
	}
}

// ---- handlers ----

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.uc.List(r.Context(), owner.From(r.Context()))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	out := make([]transactionDTO, 0, len(list))
	for _, t := range list {
		out = append(out, toDTO(t))
	}
	httpserver.OK(w, http.StatusOK, "Transactions fetched", out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	date, err := clock.ParseDate(req.Date)
	if err != nil {
		httpserver.WriteError(w, apperror.Invalid(errors.New("Date must be yyyy-mm-dd"),
			map[string]string{"date": "must be yyyy-mm-dd"}))
		return
	}
	t, err := h.uc.Create(r.Context(), owner.From(r.Context()), domain.Params{
		Type:     domain.Type(req.Type),
		Amount:   req.Amount,
		Category: req.Category,
		Date:     date,
		Note:     req.Note,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusCreated, "Transaction created", toDTO(t))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var req updateRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	patch := domain.Patch{
		Amount:   req.Amount,
		Category: req.Category,
		Note:     req.Note,
	}
	if req.Type != nil {
		typ := domain.Type(*req.Type)
		patch.Type = &typ
	}
	if req.Date != nil {
		date, err := clock.ParseDate(*req.Date)
		if err != nil {
			httpserver.WriteError(w, apperror.Invalid(errors.New("Date must be yyyy-mm-dd"),
				map[string]string{"date": "must be yyyy-mm-dd"}))
			return
		}
		patch.Date = &date
	}
	t, err := h.uc.Update(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id"), patch)
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusOK, "Transaction updated", toDTO(t))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Delete(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id")); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusOK, "Transaction deleted", nil)
}
