package handlers

import (
	"encoding/json"
	"net/http"

	"lockcenter-backend/internal/application"
	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/presentation/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ClientHandler struct {
	*BaseHandler
	service *application.ClientService
}

func NewClientHandler(service *application.ClientService) *ClientHandler {
	return &ClientHandler{
		BaseHandler: &BaseHandler{},
		service:     service,
	}
}

func (h *ClientHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input application.CreateClientInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	output, err := h.service.Create(r.Context(), claims.UserID, input)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	h.Success(w, output, http.StatusCreated)
}

func (h *ClientHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	output, err := h.service.List(r.Context(), claims.UserID, search)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	h.Success(w, output, http.StatusOK)
}

func (h *ClientHandler) ListStale(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	output, err := h.service.ListStale(r.Context(), claims.UserID)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	h.Success(w, output, http.StatusOK)
}

func (h *ClientHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	output, err := h.service.GetByID(r.Context(), claims.UserID, id)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	h.Success(w, output, http.StatusOK)
}

func (h *ClientHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	var input application.UpdateClientInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	output, err := h.service.Update(r.Context(), claims.UserID, id, input)
	if err != nil {
		h.HandleError(w, err)
		return
	}

	h.Success(w, output, http.StatusOK)
}

func (h *ClientHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.service.Delete(r.Context(), claims.UserID, id); err != nil {
		h.HandleError(w, err)
		return
	}

	h.NoContent(w)
}

func (h *ClientHandler) Reassign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		h.Error(w, "Invalid client ID", http.StatusBadRequest)
		return
	}

	var input struct {
		NewSellerID uuid.UUID `json:"newSellerId"`
		Reason      string    `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		h.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.service.Reassign(r.Context(), claims.UserID, id, input.NewSellerID, input.Reason); err != nil {
		h.HandleError(w, err)
		return
	}

	h.NoContent(w)
}

func (h *ClientHandler) HandleError(w http.ResponseWriter, err error) {
	switch err {
	case domain.ErrUnauthorized:
		h.Error(w, err.Error(), http.StatusUnauthorized)
	case domain.ErrForbidden:
		h.Error(w, err.Error(), http.StatusForbidden)
	case domain.ErrNotFound:
		h.Error(w, err.Error(), http.StatusNotFound)
	case domain.ErrValidation:
		h.Error(w, err.Error(), http.StatusBadRequest)
	default:
		h.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
