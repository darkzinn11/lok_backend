package application
import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
	"lockcenter-backend/internal/domain"
	"github.com/google/uuid"
)
const alertThresholdDays = 30
type VisitService struct {
	visitRepo  domain.VisitRepository
	userRepo   domain.UserRepository
	branchRepo domain.BranchRepository
	clientRepo domain.ClientRepository
}

type VisitAddressOutput struct {
	Street       string `json:"street"`
	Number       string `json:"number"`
	Complement   string `json:"complement,omitempty"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	UF           string `json:"uf"`
}

type VisitPhotoOutput struct {
	ID         uuid.UUID `json:"id"`
	URL        string    `json:"url"`
	FileName   string    `json:"fileName"`
	Size       int64     `json:"size"`
	UploadedAt time.Time `json:"uploadedAt"`
	PhotoType  string    `json:"photoType"`
}

type VisitLocationOutput struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracyMeters"`
	CapturedAt     string  `json:"capturedAt"`
	ReverseAddress string  `json:"reverseAddress,omitempty"`
}

type VisitOutput struct {
	ID                 uuid.UUID            `json:"id"`
	BranchID           string               `json:"branchId"`
	BranchName         string               `json:"branchName"`
	SellerID           string               `json:"sellerId"`
	SellerName         string               `json:"sellerName"`
	ClientName         string               `json:"clientName"`
	CNPJ               string               `json:"cnpj"`
	EmailClient        string               `json:"emailClient"`
	PhoneContact       string               `json:"phoneContact"`
	PhoneLandline      string               `json:"phoneLandline"`
	Address            VisitAddressOutput   `json:"address"`
	Date               string               `json:"date"`
	Subject            string               `json:"subject"`
	Conclusion         string               `json:"conclusion"`
	ArrivalTime        string               `json:"arrivalTime"`
	DepartureTime      string               `json:"departureTime"`
	KMStart            float64              `json:"kmStart"`
	KMEnd              float64              `json:"kmEnd"`
	ReceiverName       string               `json:"receiverName"`
	Notes              string               `json:"notes,omitempty"`
	ManagerObservation string               `json:"managerObservation,omitempty"`
	Status             string               `json:"status"`
	Location           *VisitLocationOutput `json:"location,omitempty"`
	Attachments        []VisitPhotoOutput   `json:"attachments"`
	CreatedAt          string               `json:"createdAt"`
}

type VisitPhotoInput struct {
	URL       string
	FileName  string
	Size      int64
	PhotoType string
}

type CreateVisitInput struct {
	ClientName    string
	CNPJ          string
	EmailClient   string
	PhoneContact  string
	PhoneLandline string
	Address       VisitAddressOutput
	Date          string
	Subject       string
	Conclusion    string
	ArrivalTime   string
	DepartureTime string
	KMStart       *float64
	KMEnd         *float64
	Notes         string
	Location      VisitLocationInput
	Attachments   []VisitPhotoInput
	Status        string // Added to differentiate Draft vs. Completed
}

type VisitLocationInput struct {
	Latitude       *float64
	Longitude      *float64
	AccuracyMeters *float64
	CapturedAt     string
	ReverseAddress string
}

type UpdateVisitInput struct {
	ClientName         string             `json:"clientName"`
	CNPJ               string             `json:"cnpj"`
	EmailClient        string             `json:"emailClient"`
	PhoneContact       string             `json:"phoneContact"`
	PhoneLandline      string             `json:"phoneLandline"`
	Address            VisitAddressOutput `json:"address"`
	Date               string             `json:"date"`
	Subject            string             `json:"subject"`
	Conclusion         string             `json:"conclusion"`
	ArrivalTime        string             `json:"arrivalTime"`
	DepartureTime      string             `json:"departureTime"`
	KMStart            *float64            `json:"kmStart"`
	KMEnd              *float64            `json:"kmEnd"`
	Notes              string             `json:"notes"`
	Status             string             `json:"status"`
	Location           VisitLocationInput `json:"location"`
	ManagerObservation string             `json:"managerObservation"`
}

type SellerKPIOutput struct {
	MonthVisits int `json:"monthVisits"`
	AlertsCount int `json:"alertsCount"`
	WeekVisits  int `json:"weekVisits"`
}

func NewVisitService(
	visitRepo domain.VisitRepository,
	userRepo domain.UserRepository,
	branchRepo domain.BranchRepository,
	clientRepo domain.ClientRepository,
) *VisitService {
	return &VisitService{
		visitRepo:  visitRepo,
		userRepo:   userRepo,
		branchRepo: branchRepo,
		clientRepo: clientRepo,
	}
}

func (s *VisitService) List(ctx context.Context, actorID uuid.UUID, filters domain.VisitFilters) ([]VisitOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	scopedFilters, err := s.scopeFilters(actor, filters)
	if err != nil {
		return nil, err
	}
	visits, err := s.visitRepo.List(ctx, scopedFilters)
	if err != nil {
		return nil, err
	}
	return s.mapVisits(ctx, visits)
}

func (s *VisitService) GetByID(ctx context.Context, actorID, visitID uuid.UUID) (*VisitOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	visit, err := s.visitRepo.GetByID(ctx, visitID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if err := s.ensureVisitAccess(ctx, actor, visit); err != nil {
		return nil, err
	}
	output, err := s.mapVisit(ctx, visit)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

func (s *VisitService) Create(ctx context.Context, actorID uuid.UUID, input CreateVisitInput) (*VisitOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	if !actor.IsSalesperson() {
		return nil, domain.ErrForbidden
	}
	normalized, err := s.normalizeCreateInput(input)
	if err != nil {
		return nil, err
	}

	// 1. Ownership and Existence Validation
	if actor.BranchID == nil {
		return nil, domain.ErrForbidden
	}
	client, err := s.clientRepo.GetByCNPJ(ctx, normalized.CNPJ, *actor.BranchID)
	if err == nil && client != nil {
		if client.SellerID != actor.ID {
			return nil, fmt.Errorf("%w: este cliente pertence a outro vendedor", domain.ErrForbidden)
		}
	}

	// 2. Status Mapping
	targetStatus := domain.StatusDraft // Default to Draft (RASCUNHO)
	inputStatus := strings.ToUpper(strings.TrimSpace(input.Status))
	if inputStatus == "ENVIADO" || inputStatus == "COMPLETED" || inputStatus == "SUBMITTED" {
		targetStatus = domain.StatusCompleted
	} else if inputStatus == "ANÁLISE" || inputStatus == "ANALISE" || inputStatus == "IN_ANALYSIS" {
		targetStatus = domain.StatusInAnalysis
	}

	// 3. Conditional Mandatory Validation
	if targetStatus == domain.StatusCompleted {
		if normalized.Notes == "" {
			return nil, fmt.Errorf("%w: as observações são obrigatórias para o envio", domain.ErrValidation)
		}
		// Client data completeness
		if normalized.ClientName == "" || normalized.CNPJ == "" || normalized.PhoneContact == "" {
			return nil, fmt.Errorf("%w: dados do cliente incompletos (nome, cnpj, telefone são obrigatórios)", domain.ErrValidation)
		}
		if normalized.Subject == "" {
			return nil, fmt.Errorf("%w: o assunto da visita é obrigatório", domain.ErrValidation)
		}
		// Proof of visit: Location OR Photos
		hasGeo := (normalized.Location.Latitude != 0 || normalized.Location.Longitude != 0)
		hasPhotos := len(normalized.Attachments) > 0
		if !hasGeo && !hasPhotos {
			return nil, fmt.Errorf("%w: Capture a localização ou adicione ao menos uma foto como comprovação da visita.", domain.ErrValidation)
		}
	}

	// 2. Draft Identification
	isDraft := (targetStatus == domain.StatusDraft)
	if isDraft {
		if normalized.ClientName == "" && normalized.CNPJ == "" {
			return nil, fmt.Errorf("%w: informe ao menos o nome do cliente ou CNPJ para salvar rascunho", domain.ErrValidation)
		}
	}

	now := time.Now().UTC()
	visit := &domain.Visit{
		ID:                 uuid.New(),
		SalespersonID:      actor.ID,
		Status:             targetStatus,
		Date:               normalized.Date,
		ClientName:         normalized.ClientName,
		ClientCNPJ:         normalized.CNPJ,
		ClientEmail:        normalized.EmailClient,
		ContactPhone:       normalized.PhoneContact,
		BranchPhone:        normalized.PhoneLandline,
		Address:            normalized.AddressJSON,
		Subject:            normalized.Subject,
		Conclusion:         normalized.Conclusion,
		ArrivalTime:        normalized.ArrivalTime,
		DepartureTime:      normalized.DepartureTime,
		KMStart:            &normalized.KMStart,
		KMEnd:              &normalized.KMEnd,
		GeoLatitude:        floatPtr(normalized.Location.Latitude),
		GeoLongitude:       floatPtr(normalized.Location.Longitude),
		GeoAccuracyMeters:  floatPtr(normalized.Location.AccuracyMeters),
		GeoCapturedAt:      timePtr(normalized.Location.CapturedAt),
		GeoReverseAddress:  normalized.Location.ReverseAddress,
		Observations:       normalized.Notes,
		ManagerObservation: "",
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.visitRepo.Create(ctx, visit); err != nil {
		return nil, err
	}
	for _, attachment := range normalized.Attachments {
		if err := s.visitRepo.AddPhoto(ctx, &domain.VisitPhoto{
			ID:        uuid.New(),
			VisitID:   visit.ID,
			BucketKey: attachment.URL,
			PublicURL: attachment.URL,
			FileName:  attachment.FileName,
			FileSize:  attachment.Size,
			PhotoType: attachment.PhotoType,
			CreatedAt: now,
		}); err != nil {
			return nil, err
		}
	}
	return s.GetByID(ctx, actorID, visit.ID)
}

func (s *VisitService) Update(ctx context.Context, actorID, visitID uuid.UUID, input UpdateVisitInput) (*VisitOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	visit, err := s.visitRepo.GetByID(ctx, visitID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// 1. Authorization
	isOwner := visit.SalespersonID == actor.ID
	isDraft := visit.Status == domain.StatusDraft || visit.Status == domain.StatusInAnalysis

	if !actor.IsManager() && !actor.IsDirector() {
		// Seller can only update if they own the visit AND it's a draft
		if !isOwner || !isDraft {
			return nil, domain.ErrForbidden
		}
	}

	if err := s.ensureVisitAccess(ctx, actor, visit); err != nil {
		return nil, err
	}

	// 2. Status Mapping
	targetStatus, err := mapInputStatus(input.Status, visit.Status, visit.Date)
	if err != nil {
		return nil, err
	}

	// 3. Conditional Mandatory Validation for "ENVIADO" (Submit)
	// If it was a draft and is now being submitted, or just being updated as submitted
	if targetStatus == domain.StatusCompleted {
		if strings.TrimSpace(input.Notes) == "" {
			return nil, fmt.Errorf("%w: as observações são obrigatórias para o envio", domain.ErrValidation)
		}
		if strings.TrimSpace(input.ClientName) == "" || strings.TrimSpace(input.CNPJ) == "" || strings.TrimSpace(input.PhoneContact) == "" {
			return nil, fmt.Errorf("%w: dados do cliente incompletos (nome, cnpj, telefone são obrigatórios)", domain.ErrValidation)
		}
		// Proof of visit
		hasGeo := derefFloat(input.Location.Latitude) != 0 || derefFloat(input.Location.Longitude) != 0
		hasPhotos := len(visit.Photos) > 0 // We use existing photos here
		if !hasGeo && !hasPhotos {
			return nil, fmt.Errorf("%w: Capture a localização ou adicione ao menos uma foto como comprovação da visita.", domain.ErrValidation)
		}
	}

	// 4. Update Fields
	// If it's a manager/director, they might only update observation/status
	// If it's a seller/draft, they update everything
	visit.ClientName = strings.TrimSpace(input.ClientName)
	visit.ClientCNPJ = strings.TrimSpace(input.CNPJ)
	visit.ClientEmail = strings.TrimSpace(input.EmailClient)
	visit.ContactPhone = normalizeDigits(input.PhoneContact)
	visit.BranchPhone = normalizeDigits(input.PhoneLandline)
	visit.Subject = strings.TrimSpace(input.Subject)
	visit.Conclusion = strings.TrimSpace(input.Conclusion)
	visit.Observations = strings.TrimSpace(input.Notes)
	visit.ManagerObservation = strings.TrimSpace(input.ManagerObservation)
	visit.Status = targetStatus

	// Minimal Draft validation: need at least Name or CNPJ to identify
	if targetStatus == domain.StatusDraft {
		if visit.ClientName == "" && visit.ClientCNPJ == "" {
			return nil, fmt.Errorf("%w: informe ao menos o nome do cliente ou CNPJ para salvar rascunho", domain.ErrValidation)
		}
	}
	visit.UpdatedAt = time.Now().UTC()

	// Update date and times if provided and valid
	if input.Date != "" {
		dateValue, err := time.Parse("2006-01-02", input.Date)
		if err == nil {
			visit.Date = dateValue
			if input.ArrivalTime != "" {
				arrival, err := combineDateAndClock(dateValue, input.ArrivalTime)
				if err == nil {
					visit.ArrivalTime = arrival
				}
			}
			if input.DepartureTime != "" {
				departure, err := combineDateAndClock(dateValue, input.DepartureTime)
				if err == nil {
					visit.DepartureTime = departure
				}
			}
		}
	}

	if input.KMStart != nil {
		visit.KMStart = input.KMStart
	}
	if input.KMEnd != nil {
		visit.KMEnd = input.KMEnd
	}

	// Update Location if provided
	if derefFloat(input.Location.Latitude) != 0 || derefFloat(input.Location.Longitude) != 0 {
		loc, err := normalizeLocation(input.Location)
		if err == nil {
			visit.GeoLatitude = &loc.Latitude
			visit.GeoLongitude = &loc.Longitude
			visit.GeoAccuracyMeters = &loc.AccuracyMeters
			visit.GeoCapturedAt = &loc.CapturedAt
			visit.GeoReverseAddress = loc.ReverseAddress
		}
	}

	// Address update
	if input.Address.Street != "" {
		addressJSON, err := marshalAddress(input.Address)
		if err == nil {
			visit.Address = addressJSON
		}
	}

	// 5. Final check for submission (already handled in step 3, but keeping it simple here)
	if targetStatus == domain.StatusCompleted {
		if visit.ClientName == "" || visit.ClientCNPJ == "" || visit.Subject == "" {
			return nil, fmt.Errorf("%w: dados obrigatórios ausentes para o envio final (Nome, CNPJ e Assunto)", domain.ErrValidation)
		}
	}

	if err := s.visitRepo.Update(ctx, visit); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, actorID, visitID)
}

func (s *VisitService) Delete(ctx context.Context, actorID, visitID uuid.UUID) error {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return domain.ErrUnauthorized
	}

	if !actor.IsManager() && !actor.IsDirector() {
		return domain.ErrForbidden
	}

	visit, err := s.visitRepo.GetByID(ctx, visitID)
	if err != nil {
		return domain.ErrNotFound
	}

	if err := s.ensureVisitAccess(ctx, actor, visit); err != nil {
		return err
	}

	if err := s.visitRepo.Delete(ctx, visitID); err != nil {
		return err
	}

	return nil
}

func (s *VisitService) GetSellerKPIs(ctx context.Context, actorID uuid.UUID) (*SellerKPIOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	if !actor.IsSalesperson() {
		return nil, domain.ErrForbidden
	}
	filters := domain.VisitFilters{SalespersonID: &actor.ID}
	visits, err := s.visitRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	weekStart := startOfWeek(now)
	weekEnd := weekStart.AddDate(0, 0, 7)
	monthVisits := 0
	alertsCount := 0
	weekVisits := 0
	for _, visit := range visits {
		if visit.Status == domain.StatusDraft {
			continue
		}
		if visit.Date.Month() == now.Month() && visit.Date.Year() == now.Year() {
			monthVisits++
		}
		if isAlertVisit(visit.Date, visit.Status, now) {
			alertsCount++
		}
		if !visit.Date.Before(weekStart) && visit.Date.Before(weekEnd) {
			weekVisits++
		}
	}
	return &SellerKPIOutput{
		MonthVisits: monthVisits,
		AlertsCount: alertsCount,
		WeekVisits:  weekVisits,
	}, nil
}

type normalizedCreateVisitInput struct {
	ClientName    string
	CNPJ          string
	EmailClient   string
	PhoneContact  string
	PhoneLandline string
	AddressJSON   string
	Date          time.Time
	Subject       string
	Conclusion    string
	ArrivalTime   *time.Time
	DepartureTime *time.Time
	KMStart       float64
	KMEnd         float64
	Notes         string
	Location      normalizedVisitLocationInput
	Attachments   []VisitPhotoInput
}

type normalizedVisitLocationInput struct {
	Latitude       float64
	Longitude      float64
	AccuracyMeters float64
	CapturedAt     time.Time
	ReverseAddress string
}

func (s *VisitService) normalizeCreateInput(input CreateVisitInput) (*normalizedCreateVisitInput, error) {
	clientName := strings.TrimSpace(input.ClientName)
	cnpj := strings.TrimSpace(input.CNPJ)
	phoneContact := normalizeDigits(input.PhoneContact)
	phoneLandline := normalizeDigits(input.PhoneLandline)
	subject := strings.TrimSpace(input.Subject)
	conclusion := strings.TrimSpace(input.Conclusion)
	notes := strings.TrimSpace(input.Notes)

	dateValue, err := time.Parse("2006-01-02", input.Date)

	if err != nil {
		return nil, fmt.Errorf("%w: invalid date", domain.ErrValidation)
	}
	isDraft := strings.ToUpper(strings.TrimSpace(input.Status)) == "RASCUNHO" || strings.ToUpper(strings.TrimSpace(input.Status)) == "DRAFT"

	arrival, err := combineDateAndClock(dateValue, input.ArrivalTime)
	if err != nil && !isDraft {
		return nil, err
	}
	departure, err := combineDateAndClock(dateValue, input.DepartureTime)
	if err != nil && !isDraft {
		return nil, err
	}
	if arrival != nil && departure != nil && !departure.After(*arrival) && !isDraft {
		return nil, fmt.Errorf("%w: departure must be after arrival", domain.ErrValidation)
	}
	if derefFloat(input.KMEnd) < derefFloat(input.KMStart) && !isDraft {
		return nil, fmt.Errorf("%w: km end must be greater than or equal to km start", domain.ErrValidation)
	}
	addressJSON, err := marshalAddress(input.Address)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid address", domain.ErrValidation)
	}
	location, err := normalizeLocation(input.Location)
	if err != nil {
		return nil, err
	}
	return &normalizedCreateVisitInput{
		ClientName:    clientName,
		CNPJ:          cnpj,
		EmailClient:   strings.TrimSpace(input.EmailClient),
		PhoneContact:  phoneContact,
		PhoneLandline: phoneLandline,
		AddressJSON:   addressJSON,
		Date:          dateValue,
		Subject:       subject,
		Conclusion:    conclusion,
		ArrivalTime:   arrival,
		DepartureTime: departure,
		KMStart:       derefFloat(input.KMStart),
		KMEnd:         derefFloat(input.KMEnd),
		Notes:         notes,
		Location:      *location,
		Attachments:   normalizeAttachments(input.Attachments),
	}, nil
}

func (s *VisitService) scopeFilters(actor *domain.User, filters domain.VisitFilters) (domain.VisitFilters, error) {
	if actor.IsSalesperson() {
		filters.SalespersonID = &actor.ID
		return filters, nil
	}
	if actor.IsManager() {
		if actor.BranchID == nil {
			return domain.VisitFilters{}, domain.ErrForbidden
		}
		filters.BranchID = actor.BranchID
	}
	return filters, nil
}

func (s *VisitService) ensureVisitAccess(ctx context.Context, actor *domain.User, visit *domain.Visit) error {
	if actor.IsDirector() {
		return nil
	}
	if actor.IsSalesperson() && visit.SalespersonID == actor.ID {
		return nil
	}
	if actor.IsManager() {
		salesperson, err := s.userRepo.GetByID(ctx, visit.SalespersonID)
		if err != nil || actor.BranchID == nil || salesperson.BranchID == nil {
			return domain.ErrForbidden
		}
		if *actor.BranchID == *salesperson.BranchID {
			return nil
		}
	}
	return domain.ErrForbidden
}

func (s *VisitService) mapVisits(ctx context.Context, visits []*domain.Visit) ([]VisitOutput, error) {
	output := make([]VisitOutput, 0, len(visits))
	for _, visit := range visits {
		item, err := s.mapVisit(ctx, visit)
		if err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, nil
}

func (s *VisitService) mapVisit(ctx context.Context, visit *domain.Visit) (VisitOutput, error) {
	seller, err := s.userRepo.GetByID(ctx, visit.SalespersonID)
	if err != nil {
		return VisitOutput{}, err
	}
	address := parseAddress(visit.Address)
	branchID := ""
	branchName := "Lokcenter - Unidade Principal"
	if seller.BranchID != nil {
		branchID = seller.BranchID.String()
		branch, err := s.branchRepo.GetByID(ctx, *seller.BranchID)
		if err == nil && branch != nil {
			branchName = branch.Name
		}
	}
	attachments := make([]VisitPhotoOutput, 0, len(visit.Photos))
	for _, photo := range visit.Photos {
		attachments = append(attachments, VisitPhotoOutput{
			ID:         photo.ID,
			URL:        photo.PublicURL,
			FileName:   photo.FileName,
			Size:       photo.FileSize,
			UploadedAt: photo.CreatedAt,
			PhotoType:  photo.PhotoType,
		})
	}
	return VisitOutput{
		ID:                 visit.ID,
		BranchID:           branchID,
		BranchName:         branchName,
		SellerID:           seller.ID.String(),
		SellerName:         seller.Name,
		ClientName:         visit.ClientName,
		CNPJ:               visit.ClientCNPJ,
		EmailClient:        visit.ClientEmail,
		PhoneContact:       visit.ContactPhone,
		PhoneLandline:      visit.BranchPhone,
		Address:            address,
		Date:               visit.Date.Format("2006-01-02"),
		Subject:            visit.Subject,
		Conclusion:         visit.Conclusion,
		ArrivalTime:        formatClock(visit.ArrivalTime),
		DepartureTime:      formatClock(visit.DepartureTime),
		KMStart:            derefFloat(visit.KMStart),
		KMEnd:              derefFloat(visit.KMEnd),
		ReceiverName:       visit.ReceiverName,
		Notes:              visit.Observations,
		ManagerObservation: visit.ManagerObservation,
		Status:             mapOutputStatus(visit.Date, visit.Status, time.Now()),
		Location:           mapLocation(visit),
		Attachments:        attachments,
		CreatedAt:          visit.CreatedAt.Format(time.RFC3339),
	}, nil
}

func mapLocation(visit *domain.Visit) *VisitLocationOutput {
	if visit.GeoLatitude == nil || visit.GeoLongitude == nil || visit.GeoAccuracyMeters == nil || visit.GeoCapturedAt == nil {
		return nil
	}
	return &VisitLocationOutput{
		Latitude:       *visit.GeoLatitude,
		Longitude:      *visit.GeoLongitude,
		AccuracyMeters: *visit.GeoAccuracyMeters,
		CapturedAt:     visit.GeoCapturedAt.Format(time.RFC3339),
		ReverseAddress: strings.TrimSpace(visit.GeoReverseAddress),
	}
}

func mapOutputStatus(date time.Time, status domain.VisitStatus, now time.Time) string {
	if isAlertVisit(date, status, now) {
		return "ALERTA"
	}
	switch status {
	case domain.StatusCompleted:
		return "ENVIADO"
	case domain.StatusPending:
		return "PENDENTE"
	case domain.StatusDraft:
		return "RASCUNHO"
	default:
		return "ANÁLISE"
	}
}

func mapInputStatus(input string, fallback domain.VisitStatus, now time.Time) (domain.VisitStatus, error) {
	value := strings.TrimSpace(strings.ToUpper(input))
	if value == "" {
		return fallback, nil
	}
	switch value {
	case "ENVIADO", "SUBMITTED", "COMPLETED":
		return domain.StatusCompleted, nil
	case "PENDENTE", "PENDING":
		return domain.StatusPending, nil
	case "ANÁLISE", "ANALISE", "IN_ANALYSIS":
		return domain.StatusInAnalysis, nil
	case "RASCUNHO", "DRAFT":
		return domain.StatusDraft, nil
	case "ALERTA":
		return domain.StatusPending, nil
	default:
		return "", fmt.Errorf("%w: invalid status", domain.ErrValidation)
	}
}

func isAlertVisit(date time.Time, status domain.VisitStatus, now time.Time) bool {
	if status == domain.StatusCompleted || status == domain.StatusDraft {
		return false
	}
	return int(now.Sub(date).Hours()/24) > alertThresholdDays
}

func combineDateAndClock(date time.Time, clock string) (*time.Time, error) {
	parts := strings.Split(strings.TrimSpace(clock), ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: invalid time", domain.ErrValidation)
	}
	value, err := time.Parse("2006-01-02 15:04", date.Format("2006-01-02")+" "+clock)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid time", domain.ErrValidation)
	}
	return &value, nil
}

func marshalAddress(address VisitAddressOutput) (string, error) {
	payload := VisitAddressOutput{
		Street:       strings.TrimSpace(address.Street),
		Number:       strings.TrimSpace(address.Number),
		Complement:   strings.TrimSpace(address.Complement),
		Neighborhood: strings.TrimSpace(address.Neighborhood),
		City:         strings.TrimSpace(address.City),
		UF:           strings.TrimSpace(address.UF),
	}
	// Relax strict check here. Validation is handled in Create/Update based on status.
	// We still want to return an empty string if nothing was provided to avoid empty JSON {}
	if payload.Street == "" && payload.Number == "" && payload.City == "" {
		return "", nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
func parseAddress(raw string) VisitAddressOutput {
	var address VisitAddressOutput
	if strings.TrimSpace(raw) == "" {
		return address
	}
	if err := json.Unmarshal([]byte(raw), &address); err == nil {
		return address
	}
	return VisitAddressOutput{Street: raw}
}

func normalizeAttachments(items []VisitPhotoInput) []VisitPhotoInput {
	output := make([]VisitPhotoInput, 0, len(items))
	for _, item := range items {
		output = append(output, VisitPhotoInput{
			URL:       strings.TrimSpace(item.URL),
			FileName:  strings.TrimSpace(item.FileName),
			Size:      item.Size,
			PhotoType: strings.TrimSpace(item.PhotoType),
		})
	}
	return output
}

func normalizeLocation(input VisitLocationInput) (*normalizedVisitLocationInput, error) {
	lat := derefFloat(input.Latitude)
	long := derefFloat(input.Longitude)
	acc := derefFloat(input.AccuracyMeters)

	if math.IsNaN(lat) || math.IsNaN(long) || math.IsNaN(acc) {
		return nil, fmt.Errorf("%w: invalid location coordinates", domain.ErrValidation)
	}

	// If everything is zero, consider it an empty location (no coordinates captured)
	if lat == 0 && long == 0 && acc == 0 {
		return &normalizedVisitLocationInput{}, nil
	}

	if lat < -90 || lat > 90 {
		return nil, fmt.Errorf("%w: latitude out of range", domain.ErrValidation)
	}
	if long < -180 || long > 180 {
		return nil, fmt.Errorf("%w: longitude out of range", domain.ErrValidation)
	}

	// Relax accuracy check as per USER REQUEST:
	// "accuracy deve ser aceito se for número positivo"
	// "Não bloquear automaticamente porque a precisão é 100m, 180m, 300m etc"
	if acc < 0 {
		acc = 0
	}

	// If we have coordinates but missing timestamp, return error
	if input.CapturedAt == "" && (lat != 0 || long != 0) {
		return nil, fmt.Errorf("%w: location timestamp is required when coordinates are provided", domain.ErrValidation)
	}

	var capturedAt time.Time
	if input.CapturedAt != "" {
		var err error
		capturedAt, err = time.Parse(time.RFC3339, strings.TrimSpace(input.CapturedAt))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid location timestamp", domain.ErrValidation)
		}
		if capturedAt.After(time.Now().UTC().Add(5 * time.Minute)) {
			return nil, fmt.Errorf("%w: location timestamp cannot be in the future", domain.ErrValidation)
		}
	}
	return &normalizedVisitLocationInput{
		Latitude:       lat,
		Longitude:      long,
		AccuracyMeters: acc,
		CapturedAt:     capturedAt.UTC(),
		ReverseAddress: strings.TrimSpace(input.ReverseAddress),
	}, nil
}

func normalizeDigits(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func formatClock(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format("15:04")
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func floatPtr(value float64) *float64 {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func startOfWeek(now time.Time) time.Time {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := now.AddDate(0, 0, -(weekday - 1))
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
}
