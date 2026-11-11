// Package handler = endpoint HTTP wishlist.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"financial-tracker/internal/shared/owner"
	"financial-tracker/internal/wishlist/domain"
	"financial-tracker/platform/httpserver"
)

// Usecase = kebutuhan handler terhadap usecase wishlist.
type Usecase interface {
	List(ctx context.Context, ownerID string) ([]domain.Item, error)
	Create(ctx context.Context, ownerID string, p domain.Params) (domain.Item, error)
	Delete(ctx context.Context, ownerID, id string) error
}

type Handler struct{ uc Usecase }

func New(uc Usecase) *Handler { return &Handler{uc: uc} }

// Routes mendaftarkan route relatif terhadap /api.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/wishlist", h.list)
	r.Post("/wishlist", h.create)
	r.Delete("/wishlist/{id}", h.delete)
}

// ---- DTO (bentuk JSON sama dengan IWishItem di josh-vault) ----

type createRequest struct {
	Name  string `json:"name"`
	Price int64  `json:"price"`
}

type itemDTO struct {
	ID        string    `json:"_id"`
	Name      string    `json:"name"`
	Price     int64     `json:"price"`
	CreatedAt time.Time `json:"createdAt"`
}

func toDTO(it domain.Item) itemDTO {
	return itemDTO{ID: it.ID, Name: it.Name, Price: it.Price, CreatedAt: it.CreatedAt.UTC()}
}

// ---- handlers ----

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.uc.List(r.Context(), owner.From(r.Context()))
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	out := make([]itemDTO, 0, len(list))
	for _, it := range list {
		out = append(out, toDTO(it))
	}
	httpserver.OK(w, http.StatusOK, "Wishlist fetched", out)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httpserver.DecodeJSON(w, r, &req); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	it, err := h.uc.Create(r.Context(), owner.From(r.Context()), domain.Params{
		Name:  req.Name,
		Price: req.Price,
	})
	if err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusCreated, "Wishlist item added", toDTO(it))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.uc.Delete(r.Context(), owner.From(r.Context()), chi.URLParam(r, "id")); err != nil {
		httpserver.WriteError(w, err)
		return
	}
	httpserver.OK(w, http.StatusOK, "Wishlist item removed", nil)
}
