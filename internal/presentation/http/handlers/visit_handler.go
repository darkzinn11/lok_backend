package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"lockcenter-backend/internal/application"
	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/presentation/http/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type VisitHandler struct {
	*BaseHandler
	visitService *application.VisitService
}

func NewVisitHandler(visitService *application.VisitService) *VisitHandler {
	return &VisitHandler{
		BaseHandler:  NewBaseHandler(),
		visitService: visitService,
	}
}

func (h *VisitHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	filters, err := parseVisitFilters(r)
	if err != nil {
		h.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	visits, svcErr := h.visitService.List(r.Context(), claims.UserID, filters)
	if svcErr != nil {
		h.handleServiceError(w, svcErr)
		return
	}

	h.Success(w, visits, http.StatusOK)
}

func (h *VisitHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	visitID, err := uuid.Parse(chi.URLParam(r, "visitID"))
	if err != nil {
		h.Error(w, "invalid visit id", http.StatusBadRequest)
		return
	}

	visit, svcErr := h.visitService.GetByID(r.Context(), claims.UserID, visitID)
	if svcErr != nil {
		h.handleServiceError(w, svcErr)
		return
	}

	h.Success(w, visit, http.StatusOK)
}

func (h *VisitHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		ClientName    string `json:"clientName"`
		CNPJ          string `json:"cnpj"`
		EmailClient   string `json:"emailClient"`
		PhoneContact  string `json:"phoneContact"`
		PhoneLandline string `json:"phoneLandline"`
		Address       struct {
			Street       string `json:"street"`
			Number       string `json:"number"`
			Complement   string `json:"complement"`
			Neighborhood string `json:"neighborhood"`
			City         string `json:"city"`
			UF           string `json:"uf"`
		} `json:"address"`
		Date          string  `json:"date"`
		Subject       string  `json:"subject"`
		Conclusion    string  `json:"conclusion"`
		ArrivalTime   string  `json:"arrivalTime"`
		DepartureTime string  `json:"departureTime"`
		KMStart       *float64 `json:"kmStart"`
		KMEnd         *float64 `json:"kmEnd"`
		Notes         string  `json:"notes"`
		Status        string  `json:"status"` // Accept status from frontend
		Location      struct {
			Latitude       *float64 `json:"latitude"`
			Longitude      *float64 `json:"longitude"`
			AccuracyMeters *float64 `json:"accuracyMeters"`
			CapturedAt     string  `json:"capturedAt"`
			ReverseAddress string  `json:"reverseAddress"`
		} `json:"location"`
		Attachments []struct {
			URL       string `json:"url"`
			FileName  string `json:"fileName"`
			Size      int64  `json:"size"`
			PhotoType string `json:"photoType"`
		} `json:"attachments"`
	}

	if err := h.Decode(r, &req); err != nil {
		h.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	attachments := make([]application.VisitPhotoInput, 0, len(req.Attachments))
	for _, item := range req.Attachments {
		attachments = append(attachments, application.VisitPhotoInput{
			URL:       item.URL,
			FileName:  item.FileName,
			Size:      item.Size,
			PhotoType: item.PhotoType,
		})
	}

	visit, svcErr := h.visitService.Create(r.Context(), claims.UserID, application.CreateVisitInput{
		ClientName:    req.ClientName,
		CNPJ:          req.CNPJ,
		EmailClient:   req.EmailClient,
		PhoneContact:  req.PhoneContact,
		PhoneLandline: req.PhoneLandline,
		Address: application.VisitAddressOutput{
			Street:       req.Address.Street,
			Number:       req.Address.Number,
			Complement:   req.Address.Complement,
			Neighborhood: req.Address.Neighborhood,
			City:         req.Address.City,
			UF:           req.Address.UF,
		},
		Date:          req.Date,
		Subject:       req.Subject,
		Conclusion:    req.Conclusion,
		ArrivalTime:   req.ArrivalTime,
		DepartureTime: req.DepartureTime,
		KMStart:       req.KMStart,
		KMEnd:         req.KMEnd,
		Notes:         req.Notes,
		Status:        req.Status,
		Location: application.VisitLocationInput{
			Latitude:       req.Location.Latitude,
			Longitude:      req.Location.Longitude,
			AccuracyMeters: req.Location.AccuracyMeters,
			CapturedAt:     req.Location.CapturedAt,
			ReverseAddress: req.Location.ReverseAddress,
		},
		Attachments: attachments,
	})
	if svcErr != nil {
		h.handleServiceError(w, svcErr)
		return
	}

	h.Success(w, visit, http.StatusCreated)
}

func (h *VisitHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	visitID, err := uuid.Parse(chi.URLParam(r, "visitID"))
	if err != nil {
		h.Error(w, "invalid visit id", http.StatusBadRequest)
		return
	}

	var req struct {
		ClientName    string `json:"clientName"`
		CNPJ          string `json:"cnpj"`
		EmailClient   string `json:"emailClient"`
		PhoneContact  string `json:"phoneContact"`
		PhoneLandline string `json:"phoneLandline"`
		Address       struct {
			Street       string `json:"street"`
			Number       string `json:"number"`
			Complement   string `json:"complement"`
			Neighborhood string `json:"neighborhood"`
			City         string `json:"city"`
			UF           string `json:"uf"`
		} `json:"address"`
		Date          string  `json:"date"`
		Subject       string  `json:"subject"`
		Conclusion    string  `json:"conclusion"`
		ArrivalTime   string  `json:"arrivalTime"`
		DepartureTime string  `json:"departureTime"`
		KMStart       *float64 `json:"kmStart"`
		KMEnd         *float64 `json:"kmEnd"`
		Notes         string  `json:"notes"`
		Status        string  `json:"status"`
		Location      struct {
			Latitude       *float64 `json:"latitude"`
			Longitude      *float64 `json:"longitude"`
			AccuracyMeters *float64 `json:"accuracyMeters"`
			CapturedAt     string  `json:"capturedAt"`
			ReverseAddress string  `json:"reverseAddress"`
		} `json:"location"`
		ManagerObservation string `json:"managerObservation"`
	}

	if err := h.Decode(r, &req); err != nil {
		h.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	visit, svcErr := h.visitService.Update(r.Context(), claims.UserID, visitID, application.UpdateVisitInput{
		ClientName:    req.ClientName,
		CNPJ:          req.CNPJ,
		EmailClient:   req.EmailClient,
		PhoneContact:  req.PhoneContact,
		PhoneLandline: req.PhoneLandline,
		Address: application.VisitAddressOutput{
			Street:       req.Address.Street,
			Number:       req.Address.Number,
			Complement:   req.Address.Complement,
			Neighborhood: req.Address.Neighborhood,
			City:         req.Address.City,
			UF:           req.Address.UF,
		},
		Date:          req.Date,
		Subject:       req.Subject,
		Conclusion:    req.Conclusion,
		ArrivalTime:   req.ArrivalTime,
		DepartureTime: req.DepartureTime,
		KMStart:       req.KMStart,
		KMEnd:         req.KMEnd,
		Notes:         req.Notes,
		Status:        req.Status,
		Location: application.VisitLocationInput{
			Latitude:       req.Location.Latitude,
			Longitude:      req.Location.Longitude,
			AccuracyMeters: req.Location.AccuracyMeters,
			CapturedAt:     req.Location.CapturedAt,
			ReverseAddress: req.Location.ReverseAddress,
		},
		ManagerObservation: req.ManagerObservation,
	})
	if svcErr != nil {
		h.handleServiceError(w, svcErr)
		return
	}

	h.Success(w, visit, http.StatusOK)
}

func (h *VisitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	visitID, err := uuid.Parse(chi.URLParam(r, "visitID"))
	if err != nil {
		h.Error(w, "invalid visit id", http.StatusBadRequest)
		return
	}

	if svcErr := h.visitService.Delete(r.Context(), claims.UserID, visitID); svcErr != nil {
		h.handleServiceError(w, svcErr)
		return
	}

	h.Success(w, nil, http.StatusNoContent)
}

func (h *VisitHandler) SellerKPIs(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserClaims(r.Context())
	if !ok {
		h.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	kpis, svcErr := h.visitService.GetSellerKPIs(r.Context(), claims.UserID)
	if svcErr != nil {
		h.handleServiceError(w, svcErr)
		return
	}

	h.Success(w, kpis, http.StatusOK)
}

func (h *VisitHandler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		h.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, domain.ErrUnauthorized):
		h.Error(w, "unauthorized", http.StatusUnauthorized)
	case errors.Is(err, domain.ErrNotFound):
		h.Error(w, "resource not found", http.StatusNotFound)
	case errors.Is(err, domain.ErrValidation):
		h.Error(w, err.Error(), http.StatusBadRequest)
	default:
		h.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func parseVisitFilters(r *http.Request) (domain.VisitFilters, error) {
	var filters domain.VisitFilters

	filters.Search = r.URL.Query().Get("search")
	filters.Subject = r.URL.Query().Get("subject")
	filters.Conclusion = r.URL.Query().Get("conclusion")
	filters.OnlyAlerts = strings.EqualFold(r.URL.Query().Get("only_alerts"), "true")

	if sellerID := strings.TrimSpace(r.URL.Query().Get("seller_id")); sellerID != "" {
		parsedID, err := uuid.Parse(sellerID)
		if err != nil {
			return domain.VisitFilters{}, errors.New("invalid seller id")
		}
		filters.SalespersonID = &parsedID
	}

	if branchID := strings.TrimSpace(r.URL.Query().Get("branch_id")); branchID != "" {
		parsedID, err := uuid.Parse(branchID)
		if err != nil {
			return domain.VisitFilters{}, errors.New("invalid branch id")
		}
		filters.BranchID = &parsedID
	}

	if status := strings.TrimSpace(r.URL.Query().Get("status")); status != "" {
		mapped, err := mapQueryStatus(status)
		if err != nil {
			return domain.VisitFilters{}, err
		}
		filters.Status = &mapped
	}

	if date := strings.TrimSpace(r.URL.Query().Get("date")); date != "" {
		parsedDate, err := time.Parse("2006-01-02", date)
		if err != nil {
			return domain.VisitFilters{}, errors.New("invalid date")
		}
		filters.Date = &parsedDate
	}

	return filters, nil
}

func mapQueryStatus(value string) (domain.VisitStatus, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ENVIADO", "SUBMITTED", "COMPLETED":
		return domain.StatusCompleted, nil
	case "PENDENTE", "PENDING", "ALERTA":
		return domain.StatusPending, nil
	case "ANÁLISE", "ANALISE", "IN_ANALYSIS":
		return domain.StatusInAnalysis, nil
	case "RASCUNHO", "DRAFT":
		return domain.StatusDraft, nil
	default:
		return "", errors.New("invalid status")
	}
}
