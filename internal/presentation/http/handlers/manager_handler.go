package handlers

import (
	"errors"
	"net/http"

	"lockcenter-backend/internal/application"
	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/presentation/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ManagerHandler struct {
	*BaseHandler
	managerService *application.ManagerService
}

func NewManagerHandler(managerService *application.ManagerService) *ManagerHandler {
	return &ManagerHandler{
		BaseHandler:    NewBaseHandler(),
		managerService: managerService,
	}
}

func (h *ManagerHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	filters, err := parseManagerFilters(r)
	if err != nil {
		h.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	managers, svcErr := h.managerService.List(r.Context(), claims.UserID, filters)
	if svcErr != nil {
		h.handleManagerServiceError(w, svcErr)
		return
	}

	h.Success(w, managers, http.StatusOK)
}

func (h *ManagerHandler) Stats(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	stats, svcErr := h.managerService.GetStats(r.Context(), claims.UserID)
	if svcErr != nil {
		h.handleManagerServiceError(w, svcErr)
		return
	}

	h.Success(w, stats, http.StatusOK)
}

func (h *ManagerHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	managerID, err := uuid.Parse(chi.URLParam(r, "managerID"))
	if err != nil {
		h.Error(w, "invalid manager id", http.StatusBadRequest)
		return
	}

	manager, svcErr := h.managerService.GetByID(r.Context(), claims.UserID, managerID)
	if svcErr != nil {
		h.handleManagerServiceError(w, svcErr)
		return
	}

	h.Success(w, manager, http.StatusOK)
}

func (h *ManagerHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name      string `json:"name" validate:"required,min=3,max=255"`
		Email     string `json:"email" validate:"required,email,max=255"`
		Phone     string `json:"phone" validate:"required,min=10,max=20"`
		CPF       string `json:"cpf" validate:"required,min=11,max=14"`
		PhotoURL  string `json:"photo_url"`
		BirthDate string `json:"birth_date" validate:"required"`
		BranchID  string `json:"branch_id" validate:"required"`
		Status    string `json:"status"`
		Password  string `json:"password"`
	}

	if err := h.Decode(r, &req); err != nil {
		h.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	branchID, err := uuid.Parse(req.BranchID)
	if err != nil {
		h.Error(w, "invalid branch id", http.StatusBadRequest)
		return
	}

	result, svcErr := h.managerService.Create(r.Context(), claims.UserID, application.CreateSellerInput{
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		CPF:       req.CPF,
		PhotoURL:  req.PhotoURL,
		BirthDate: req.BirthDate,
		BranchID:  branchID,
		Status:    domain.UserStatus(req.Status),
		Password:  req.Password,
	})
	if svcErr != nil {
		h.handleManagerServiceError(w, svcErr)
		return
	}

	h.Success(w, result, http.StatusCreated)
}

func (h *ManagerHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	managerID, err := uuid.Parse(chi.URLParam(r, "managerID"))
	if err != nil {
		h.Error(w, "invalid manager id", http.StatusBadRequest)
		return
	}

	var req struct {
		Name      string `json:"name" validate:"required,min=3,max=255"`
		Email     string `json:"email" validate:"required,email,max=255"`
		Phone     string `json:"phone" validate:"required,min=10,max=20"`
		CPF       string `json:"cpf" validate:"required,min=11,max=14"`
		PhotoURL  string `json:"photo_url"`
		BirthDate string `json:"birth_date" validate:"required"`
		BranchID  string `json:"branch_id" validate:"required"`
		Status    string `json:"status" validate:"required"`
	}

	if err := h.Decode(r, &req); err != nil {
		h.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	branchID, err := uuid.Parse(req.BranchID)
	if err != nil {
		h.Error(w, "invalid branch id", http.StatusBadRequest)
		return
	}

	manager, svcErr := h.managerService.Update(r.Context(), claims.UserID, managerID, application.UpdateSellerInput{
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		CPF:       req.CPF,
		PhotoURL:  req.PhotoURL,
		BirthDate: req.BirthDate,
		BranchID:  branchID,
		Status:    domain.UserStatus(req.Status),
	})
	if svcErr != nil {
		h.handleManagerServiceError(w, svcErr)
		return
	}

	h.Success(w, manager, http.StatusOK)
}

func (h *ManagerHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	managerID, err := uuid.Parse(chi.URLParam(r, "managerID"))
	if err != nil {
		h.Error(w, "invalid manager id", http.StatusBadRequest)
		return
	}

	if svcErr := h.managerService.Delete(r.Context(), claims.UserID, managerID); svcErr != nil {
		h.handleManagerServiceError(w, svcErr)
		return
	}

	h.Success(w, map[string]string{"message": "manager deleted successfully"}, http.StatusOK)
}

func (h *ManagerHandler) handleManagerServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		h.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, domain.ErrNotFound):
		h.Error(w, "resource not found", http.StatusNotFound)
	case errors.Is(err, domain.ErrConflict):
		h.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, domain.ErrValidation):
		h.Error(w, err.Error(), http.StatusBadRequest)
	default:
		h.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func parseManagerFilters(r *http.Request) (domain.SellerFilters, error) {
	var filters domain.SellerFilters

	filters.Search = r.URL.Query().Get("search")

	if branchID := r.URL.Query().Get("branch_id"); branchID != "" {
		parsedBranchID, err := uuid.Parse(branchID)
		if err != nil {
			return domain.SellerFilters{}, errors.New("invalid branch id")
		}
		filters.BranchID = &parsedBranchID
	}

	if status := r.URL.Query().Get("status"); status != "" {
		normalized := domain.UserStatus(status)
		filters.Status = &normalized
	}

	return filters, nil
}
