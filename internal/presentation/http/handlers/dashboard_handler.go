package handlers

import (
	"errors"
	"net/http"
	"time"

	"lockcenter-backend/internal/application"
	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/presentation/http/middleware"

	"github.com/google/uuid"
)

type DashboardHandler struct {
	*BaseHandler
	dashboardService *application.DashboardService
}

func NewDashboardHandler(dashboardService *application.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		BaseHandler:      NewBaseHandler(),
		dashboardService: dashboardService,
	}
}

func (h *DashboardHandler) Overview(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	startDate, endDate, branchID, err := parseDashboardRange(r)
	if err != nil {
		h.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	output, svcErr := h.dashboardService.GetOverview(r.Context(), claims.UserID, application.DashboardRange{
		StartDate: startDate,
		EndDate:   endDate,
		BranchID:  branchID,
	})
	if svcErr != nil {
		switch {
		case errors.Is(svcErr, domain.ErrUnauthorized):
			h.Error(w, "unauthorized", http.StatusUnauthorized)
		case errors.Is(svcErr, domain.ErrForbidden):
			h.Error(w, "forbidden", http.StatusForbidden)
		case errors.Is(svcErr, domain.ErrValidation):
			h.Error(w, "invalid dashboard date range", http.StatusBadRequest)
		default:
			h.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	h.Success(w, output, http.StatusOK)
}

func (h *DashboardHandler) SellerReport(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	startDate, endDate, branchID, err := parseDashboardRange(r)
	if err != nil {
		h.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	output, svcErr := h.dashboardService.GetSellerReport(r.Context(), claims.UserID, application.DashboardRange{
		StartDate: startDate,
		EndDate:   endDate,
		BranchID:  branchID,
	})
	if svcErr != nil {
		switch {
		case errors.Is(svcErr, domain.ErrUnauthorized):
			h.Error(w, "unauthorized", http.StatusUnauthorized)
		case errors.Is(svcErr, domain.ErrForbidden):
			h.Error(w, "forbidden", http.StatusForbidden)
		case errors.Is(svcErr, domain.ErrValidation):
			h.Error(w, "invalid dashboard date range", http.StatusBadRequest)
		default:
			h.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	h.Success(w, output, http.StatusOK)
}

func parseDashboardRange(r *http.Request) (time.Time, time.Time, *uuid.UUID, error) {
	now := time.Now().UTC()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	if value := r.URL.Query().Get("start_date"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, time.Time{}, nil, errors.New("invalid start_date")
		}
		startDate = parsed.UTC()
	}

	if value := r.URL.Query().Get("end_date"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			return time.Time{}, time.Time{}, nil, errors.New("invalid end_date")
		}
		endDate = parsed.UTC()
	}

	var branchID *uuid.UUID
	if value := r.URL.Query().Get("branch_id"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return time.Time{}, time.Time{}, nil, errors.New("invalid branch_id")
		}
		branchID = &parsed
	}

	return startDate, endDate, branchID, nil
}
