package application

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/infrastructure/security"

	"github.com/google/uuid"
)

type SellerService struct {
	sellerRepo domain.SellerRepository
	userRepo   domain.UserRepository
	branchRepo domain.BranchRepository
	hasher     security.PasswordHasher
}

type SellerOutput struct {
	ID         uuid.UUID         `json:"id"`
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	Phone      string            `json:"phone"`
	CPF        string            `json:"cpf"`
	PhotoURL   string            `json:"photo_url,omitempty"`
	BirthDate  string            `json:"birth_date,omitempty"`
	Role       domain.Role       `json:"role"`
	Status     domain.UserStatus `json:"status"`
	BranchID   *uuid.UUID        `json:"branch_id,omitempty"`
	BranchName string            `json:"branch_name,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
}

type CreateSellerInput struct {
	Name      string
	Email     string
	Phone     string
	CPF       string
	PhotoURL  string
	BirthDate string
	BranchID  uuid.UUID
	Status    domain.UserStatus
	Password  string
}

type CreateSellerResult struct {
	Seller            SellerOutput `json:"seller"`
	TemporaryPassword string       `json:"temporary_password,omitempty"`
}

type UpdateSellerInput struct {
	Name      string
	Email     string
	Phone     string
	CPF       string
	PhotoURL  string
	BirthDate string
	BranchID  uuid.UUID
	Status    domain.UserStatus
}

func NewSellerService(
	sellerRepo domain.SellerRepository,
	userRepo domain.UserRepository,
	branchRepo domain.BranchRepository,
	hasher security.PasswordHasher,
) *SellerService {
	return &SellerService{
		sellerRepo: sellerRepo,
		userRepo:   userRepo,
		branchRepo: branchRepo,
		hasher:     hasher,
	}
}

func (s *SellerService) List(ctx context.Context, actorID uuid.UUID, filters domain.SellerFilters) ([]SellerOutput, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.CanManageSellers() {
		return nil, domain.ErrForbidden
	}

	scopedFilters, err := s.scopeFilters(actor, filters)
	if err != nil {
		return nil, err
	}

	sellers, err := s.sellerRepo.List(ctx, scopedFilters)
	if err != nil {
		return nil, err
	}

	return s.mapSellers(ctx, sellers)
}

func (s *SellerService) GetByID(ctx context.Context, actorID, sellerID uuid.UUID) (*SellerOutput, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.CanManageSellers() {
		return nil, domain.ErrForbidden
	}

	seller, err := s.sellerRepo.GetByID(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("%w: seller not found", domain.ErrNotFound)
	}

	if err := s.ensureBranchAccess(actor, seller); err != nil {
		return nil, err
	}

	output, err := s.mapSeller(ctx, seller)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

func (s *SellerService) Create(ctx context.Context, actorID uuid.UUID, input CreateSellerInput) (*CreateSellerResult, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.CanManageSellers() {
		return nil, domain.ErrForbidden
	}

	normalizedInput, err := s.normalizeCreateInput(ctx, actor, input)
	if err != nil {
		return nil, err
	}

	if err := s.ensureUniqueSeller(ctx, normalizedInput.Email, normalizedInput.Phone, normalizedInput.CPF, nil); err != nil {
		return nil, err
	}

	password := normalizedInput.Password
	temporaryPassword := ""
	if password == "" {
		password, err = generateTemporaryPassword()
		if err != nil {
			return nil, err
		}
		temporaryPassword = password
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	seller := &domain.User{
		ID:                 uuid.New(),
		Name:               normalizedInput.Name,
		Email:              normalizedInput.Email,
		Phone:              normalizedInput.Phone,
		CPF:                normalizedInput.CPF,
		PhotoURL:           normalizedInput.PhotoURL,
		BirthDate:          parseDatePointer(normalizedInput.BirthDate),
		PasswordHash:       passwordHash,
		Role:               domain.RoleSalesperson,
		Status:             normalizedInput.Status,
		MustChangePassword: true,
		BranchID:           &normalizedInput.BranchID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.sellerRepo.Create(ctx, seller); err != nil {
		return nil, err
	}

	output, err := s.mapSeller(ctx, seller)
	if err != nil {
		return nil, err
	}

	return &CreateSellerResult{
		Seller:            output,
		TemporaryPassword: temporaryPassword,
	}, nil
}

func (s *SellerService) Update(ctx context.Context, actorID, sellerID uuid.UUID, input UpdateSellerInput) (*SellerOutput, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.CanManageSellers() {
		return nil, domain.ErrForbidden
	}

	seller, err := s.sellerRepo.GetByID(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("%w: seller not found", domain.ErrNotFound)
	}

	if err := s.ensureBranchAccess(actor, seller); err != nil {
		return nil, err
	}

	normalizedInput, err := s.normalizeUpdateInput(ctx, actor, seller, input)
	if err != nil {
		return nil, err
	}

	if err := s.ensureUniqueSeller(ctx, normalizedInput.Email, normalizedInput.Phone, normalizedInput.CPF, &sellerID); err != nil {
		return nil, err
	}

	seller.Name = normalizedInput.Name
	seller.Email = normalizedInput.Email
	seller.Phone = normalizedInput.Phone
	seller.CPF = normalizedInput.CPF
	seller.PhotoURL = normalizedInput.PhotoURL
	seller.BirthDate = parseDatePointer(normalizedInput.BirthDate)
	seller.Status = normalizedInput.Status
	seller.BranchID = &normalizedInput.BranchID
	seller.UpdatedAt = time.Now().UTC()

	if err := s.sellerRepo.Update(ctx, seller); err != nil {
		return nil, err
	}

	output, err := s.mapSeller(ctx, seller)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

func (s *SellerService) Delete(ctx context.Context, actorID, sellerID uuid.UUID) error {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return err
	}

	if !actor.CanManageSellers() {
		return domain.ErrForbidden
	}

	seller, err := s.sellerRepo.GetByID(ctx, sellerID)
	if err != nil {
		return fmt.Errorf("%w: seller not found", domain.ErrNotFound)
	}

	if err := s.ensureBranchAccess(actor, seller); err != nil {
		return err
	}

	return s.sellerRepo.Delete(ctx, sellerID)
}

func (s *SellerService) loadActor(ctx context.Context, actorID uuid.UUID) (*domain.User, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	return actor, nil
}

func (s *SellerService) scopeFilters(actor *domain.User, filters domain.SellerFilters) (domain.SellerFilters, error) {
	if actor.IsManager() {
		if actor.BranchID == nil {
			return domain.SellerFilters{}, fmt.Errorf("%w: manager without branch", domain.ErrForbidden)
		}
		filters.BranchID = actor.BranchID
	}

	return filters, nil
}

func (s *SellerService) ensureBranchAccess(actor, seller *domain.User) error {
	if actor.IsDirector() {
		return nil
	}

	if actor.IsManager() && actor.BranchID != nil && seller.BranchID != nil && *actor.BranchID == *seller.BranchID {
		return nil
	}

	return domain.ErrForbidden
}

func (s *SellerService) normalizeCreateInput(ctx context.Context, actor *domain.User, input CreateSellerInput) (CreateSellerInput, error) {
	normalized := CreateSellerInput{
		Name:      strings.TrimSpace(input.Name),
		Email:     strings.ToLower(strings.TrimSpace(input.Email)),
		Phone:     normalizePhone(input.Phone),
		CPF:       normalizeCPF(input.CPF),
		PhotoURL:  normalizePhotoURL(input.PhotoURL),
		BirthDate: strings.TrimSpace(input.BirthDate),
		BranchID:  input.BranchID,
		Status:    input.Status,
		Password:  strings.TrimSpace(input.Password),
	}

	if normalized.Status == "" {
		normalized.Status = domain.UserStatusActive
	}

	if actor.IsManager() {
		if actor.BranchID == nil {
			return CreateSellerInput{}, domain.ErrForbidden
		}
		normalized.BranchID = *actor.BranchID
	}

	if err := s.validateSellerInput(ctx, normalized.Name, normalized.Email, normalized.Phone, normalized.CPF, normalized.BirthDate, normalized.BranchID, normalized.Status); err != nil {
		return CreateSellerInput{}, err
	}

	return normalized, nil
}

func (s *SellerService) normalizeUpdateInput(ctx context.Context, actor, current *domain.User, input UpdateSellerInput) (UpdateSellerInput, error) {
	normalized := UpdateSellerInput{
		Name:      strings.TrimSpace(input.Name),
		Email:     strings.ToLower(strings.TrimSpace(input.Email)),
		Phone:     normalizePhone(input.Phone),
		CPF:       normalizeCPF(input.CPF),
		PhotoURL:  normalizePhotoURL(input.PhotoURL),
		BirthDate: strings.TrimSpace(input.BirthDate),
		BranchID:  input.BranchID,
		Status:    input.Status,
	}

	if actor.IsManager() {
		if actor.BranchID == nil {
			return UpdateSellerInput{}, domain.ErrForbidden
		}
		normalized.BranchID = *actor.BranchID
	}

	if normalized.BirthDate == "" && current.BirthDate != nil {
		normalized.BirthDate = current.BirthDate.Format("2006-01-02")
	}

	if normalized.Status == "" {
		normalized.Status = current.Status
	}

	if err := s.validateSellerInput(ctx, normalized.Name, normalized.Email, normalized.Phone, normalized.CPF, normalized.BirthDate, normalized.BranchID, normalized.Status); err != nil {
		return UpdateSellerInput{}, err
	}

	return normalized, nil
}

func (s *SellerService) validateSellerInput(ctx context.Context, name, email, phone, cpf, birthDate string, branchID uuid.UUID, status domain.UserStatus) error {
	switch {
	case name == "":
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	case email == "":
		return fmt.Errorf("%w: email is required", domain.ErrValidation)
	case phone == "":
		return fmt.Errorf("%w: phone is required", domain.ErrValidation)
	case cpf == "":
		return fmt.Errorf("%w: cpf is required", domain.ErrValidation)
	case len(cpf) != 11:
		return fmt.Errorf("%w: invalid cpf", domain.ErrValidation)
	case birthDate == "":
		return fmt.Errorf("%w: birth date is required", domain.ErrValidation)
	case branchID == uuid.Nil:
		return fmt.Errorf("%w: branch is required", domain.ErrValidation)
	case status != domain.UserStatusActive && status != domain.UserStatusInactive && status != domain.UserStatusBlocked:
		return fmt.Errorf("%w: invalid status", domain.ErrValidation)
	}

	parsedBirthDate, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		return fmt.Errorf("%w: invalid birth date", domain.ErrValidation)
	}

	if parsedBirthDate.After(time.Now()) {
		return fmt.Errorf("%w: invalid birth date", domain.ErrValidation)
	}

	if _, err := s.branchRepo.GetByID(ctx, branchID); err != nil {
		return fmt.Errorf("%w: branch not found", domain.ErrValidation)
	}

	return nil
}

func (s *SellerService) ensureUniqueSeller(ctx context.Context, email, phone, cpf string, excludeID *uuid.UUID) error {
	emailExists, err := s.sellerRepo.ExistsByEmail(ctx, email, excludeID)
	if err != nil {
		return err
	}
	if emailExists {
		return fmt.Errorf("%w: email already in use", domain.ErrConflict)
	}

	phoneExists, err := s.sellerRepo.ExistsByPhone(ctx, phone, excludeID)
	if err != nil {
		return err
	}
	if phoneExists {
		return fmt.Errorf("%w: phone already in use", domain.ErrConflict)
	}

	cpfExists, err := s.sellerRepo.ExistsByCPF(ctx, cpf, excludeID)
	if err != nil {
		return err
	}
	if cpfExists {
		return fmt.Errorf("%w: cpf already in use", domain.ErrConflict)
	}

	return nil
}

func (s *SellerService) mapSellers(ctx context.Context, sellers []*domain.User) ([]SellerOutput, error) {
	output := make([]SellerOutput, 0, len(sellers))
	for _, seller := range sellers {
		item, err := s.mapSeller(ctx, seller)
		if err != nil {
			return nil, err
		}
		output = append(output, item)
	}

	return output, nil
}

func (s *SellerService) mapSeller(ctx context.Context, seller *domain.User) (SellerOutput, error) {
	item := SellerOutput{
		ID:        seller.ID,
		Name:      seller.Name,
		Email:     seller.Email,
		Phone:     seller.Phone,
		CPF:       seller.CPF,
		PhotoURL:  seller.PhotoURL,
		Role:      seller.Role,
		Status:    seller.Status,
		BranchID:  seller.BranchID,
		CreatedAt: seller.CreatedAt,
		UpdatedAt: seller.UpdatedAt,
	}

	if seller.BranchID != nil {
		branch, err := s.branchRepo.GetByID(ctx, *seller.BranchID)
		if err == nil && branch != nil {
			item.BranchName = branch.Name
		}
	}

	if seller.BirthDate != nil {
		item.BirthDate = seller.BirthDate.Format("2006-01-02")
	}

	return item, nil
}

func normalizePhone(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

func normalizeCPF(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

func normalizePhotoURL(value string) string {
	return strings.TrimSpace(value)
}

func parseDatePointer(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil
	}

	return &parsed
}

func generateTemporaryPassword() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%"
	const size = 12

	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}

	var builder strings.Builder
	builder.Grow(size)
	for _, item := range buffer {
		builder.WriteByte(alphabet[int(item)%len(alphabet)])
	}

	return builder.String(), nil
}
