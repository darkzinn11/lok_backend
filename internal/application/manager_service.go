package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/infrastructure/security"

	"github.com/google/uuid"
)

type ManagerService struct {
	managerRepo domain.SellerRepository
	userRepo    domain.UserRepository
	branchRepo  domain.BranchRepository
	hasher      security.PasswordHasher
}

type ManagerStatsOutput struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Blocked  int `json:"blocked"`
	Inactive int `json:"inactive"`
}

func NewManagerService(
	managerRepo domain.SellerRepository,
	userRepo domain.UserRepository,
	branchRepo domain.BranchRepository,
	hasher security.PasswordHasher,
) *ManagerService {
	return &ManagerService{
		managerRepo: managerRepo,
		userRepo:    userRepo,
		branchRepo:  branchRepo,
		hasher:      hasher,
	}
}

func (s *ManagerService) List(ctx context.Context, actorID uuid.UUID, filters domain.SellerFilters) ([]SellerOutput, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.IsDirector() {
		return nil, domain.ErrForbidden
	}

	managers, err := s.managerRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Fetch all branches in bulk to avoid N+1 queries
	branchMap := make(map[uuid.UUID]string)
	branches, err := s.branchRepo.List(ctx)
	if err == nil {
		for _, b := range branches {
			branchMap[b.ID] = b.Name
		}
	}

	return s.mapManagers(ctx, managers, branchMap)
}

func (s *ManagerService) GetStats(ctx context.Context, actorID uuid.UUID) (*ManagerStatsOutput, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.IsDirector() {
		return nil, domain.ErrForbidden
	}

	active := domain.UserStatusActive
	activeCount, _ := s.managerRepo.Count(ctx, domain.SellerFilters{Status: &active})

	inactive := domain.UserStatusInactive
	inactiveCount, _ := s.managerRepo.Count(ctx, domain.SellerFilters{Status: &inactive})

	blocked := domain.UserStatusBlocked
	blockedCount, _ := s.managerRepo.Count(ctx, domain.SellerFilters{Status: &blocked})

	return &ManagerStatsOutput{
		Total:    activeCount + inactiveCount + blockedCount,
		Active:   activeCount,
		Inactive: inactiveCount,
		Blocked:  blockedCount,
	}, nil
}

func (s *ManagerService) GetByID(ctx context.Context, actorID, managerID uuid.UUID) (*SellerOutput, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.IsDirector() {
		return nil, domain.ErrForbidden
	}

	manager, err := s.managerRepo.GetByID(ctx, managerID)
	if err != nil {
		return nil, fmt.Errorf("%w: manager not found", domain.ErrNotFound)
	}

	output, err := s.mapManager(ctx, manager, nil)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

func (s *ManagerService) Create(ctx context.Context, actorID uuid.UUID, input CreateSellerInput) (*CreateSellerResult, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.IsDirector() {
		return nil, domain.ErrForbidden
	}

	normalized := s.normalizeInput(input)

	if err := s.validateInput(ctx, normalized); err != nil {
		return nil, err
	}

	if err := s.ensureUnique(ctx, normalized.Email, normalized.Phone, normalized.CPF, nil); err != nil {
		return nil, err
	}

	password := normalized.Password
	if password == "" {
		return nil, fmt.Errorf("%w: password is required", domain.ErrValidation)
	}

	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	manager := &domain.User{
		ID:                 uuid.New(),
		Name:               normalized.Name,
		Email:              normalized.Email,
		Phone:              normalized.Phone,
		CPF:                normalized.CPF,
		PhotoURL:           normalized.PhotoURL,
		BirthDate:          parseDatePointer(normalized.BirthDate),
		PasswordHash:       passwordHash,
		Role:               domain.RoleManager,
		Status:             normalized.Status,
		MustChangePassword: false,
		BranchID:           &normalized.BranchID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.managerRepo.Create(ctx, manager); err != nil {
		return nil, err
	}

	output, err := s.mapManager(ctx, manager, nil)
	if err != nil {
		return nil, err
	}

	return &CreateSellerResult{
		Seller:            output,
		TemporaryPassword: "",
	}, nil
}

func (s *ManagerService) UpdatePassword(ctx context.Context, actorID, targetUserID uuid.UUID, newPassword string) error {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return err
	}

	if !actor.IsDirector() {
		return domain.ErrForbidden
	}

	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	user, err := s.managerRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return fmt.Errorf("%w: user not found", domain.ErrNotFound)
	}

	hash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return err
	}

	user.PasswordHash = hash
	user.MustChangePassword = false
	user.UpdatedAt = time.Now().UTC()

	return s.managerRepo.Update(ctx, user)
}

func (s *ManagerService) Update(ctx context.Context, actorID, managerID uuid.UUID, input UpdateSellerInput) (*SellerOutput, error) {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return nil, err
	}

	if !actor.IsDirector() {
		return nil, domain.ErrForbidden
	}

	manager, err := s.managerRepo.GetByID(ctx, managerID)
	if err != nil {
		return nil, fmt.Errorf("%w: manager not found", domain.ErrNotFound)
	}

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

	if normalized.BirthDate == "" && manager.BirthDate != nil {
		normalized.BirthDate = manager.BirthDate.Format("2006-01-02")
	}

	if normalized.Status == "" {
		normalized.Status = manager.Status
	}

	if err := s.validateUpdateInput(ctx, normalized); err != nil {
		return nil, err
	}

	if err := s.ensureUnique(ctx, normalized.Email, normalized.Phone, normalized.CPF, &managerID); err != nil {
		return nil, err
	}

	manager.Name = normalized.Name
	manager.Email = normalized.Email
	manager.Phone = normalized.Phone
	manager.CPF = normalized.CPF
	manager.PhotoURL = normalized.PhotoURL
	manager.BirthDate = parseDatePointer(normalized.BirthDate)
	manager.Status = normalized.Status
	manager.BranchID = &normalized.BranchID
	manager.UpdatedAt = time.Now().UTC()

	if err := s.managerRepo.Update(ctx, manager); err != nil {
		return nil, err
	}

	output, err := s.mapManager(ctx, manager, nil)
	if err != nil {
		return nil, err
	}

	return &output, nil
}

func (s *ManagerService) Delete(ctx context.Context, actorID, managerID uuid.UUID) error {
	actor, err := s.loadActor(ctx, actorID)
	if err != nil {
		return err
	}

	if !actor.IsDirector() {
		return domain.ErrForbidden
	}

	if _, err := s.managerRepo.GetByID(ctx, managerID); err != nil {
		return fmt.Errorf("%w: manager not found", domain.ErrNotFound)
	}

	return s.managerRepo.Delete(ctx, managerID)
}

func (s *ManagerService) loadActor(ctx context.Context, actorID uuid.UUID) (*domain.User, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}
	return actor, nil
}

func (s *ManagerService) normalizeInput(input CreateSellerInput) CreateSellerInput {
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

	return normalized
}

func (s *ManagerService) validateInput(ctx context.Context, input CreateSellerInput) error {
	switch {
	case input.Name == "":
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	case input.Email == "":
		return fmt.Errorf("%w: email is required", domain.ErrValidation)
	case input.Phone == "":
		return fmt.Errorf("%w: phone is required", domain.ErrValidation)
	case input.CPF == "":
		return fmt.Errorf("%w: cpf is required", domain.ErrValidation)
	case len(input.CPF) != 11:
		return fmt.Errorf("%w: invalid cpf", domain.ErrValidation)
	case input.BirthDate == "":
		return fmt.Errorf("%w: birth date is required", domain.ErrValidation)
	case input.BranchID == uuid.Nil:
		return fmt.Errorf("%w: branch is required", domain.ErrValidation)
	}

	if _, err := time.Parse("2006-01-02", input.BirthDate); err != nil {
		return fmt.Errorf("%w: invalid birth date", domain.ErrValidation)
	}

	if _, err := s.branchRepo.GetByID(ctx, input.BranchID); err != nil {
		return fmt.Errorf("%w: branch not found", domain.ErrValidation)
	}

	return nil
}

func (s *ManagerService) validateUpdateInput(ctx context.Context, input UpdateSellerInput) error {
	switch {
	case input.Name == "":
		return fmt.Errorf("%w: name is required", domain.ErrValidation)
	case input.Email == "":
		return fmt.Errorf("%w: email is required", domain.ErrValidation)
	case input.Phone == "":
		return fmt.Errorf("%w: phone is required", domain.ErrValidation)
	case input.CPF == "":
		return fmt.Errorf("%w: cpf is required", domain.ErrValidation)
	case input.BranchID == uuid.Nil:
		return fmt.Errorf("%w: branch is required", domain.ErrValidation)
	}

	if input.BirthDate != "" {
		if _, err := time.Parse("2006-01-02", input.BirthDate); err != nil {
			return fmt.Errorf("%w: invalid birth date", domain.ErrValidation)
		}
	}

	if _, err := s.branchRepo.GetByID(ctx, input.BranchID); err != nil {
		return fmt.Errorf("%w: branch not found", domain.ErrValidation)
	}

	return nil
}

func (s *ManagerService) ensureUnique(ctx context.Context, email, phone, cpf string, excludeID *uuid.UUID) error {
	emailExists, err := s.managerRepo.ExistsByEmail(ctx, email, excludeID)
	if err != nil {
		return err
	}
	if emailExists {
		return fmt.Errorf("%w: email already in use", domain.ErrConflict)
	}

	phoneExists, err := s.managerRepo.ExistsByPhone(ctx, phone, excludeID)
	if err != nil {
		return err
	}
	if phoneExists {
		return fmt.Errorf("%w: phone already in use", domain.ErrConflict)
	}

	cpfExists, err := s.managerRepo.ExistsByCPF(ctx, cpf, excludeID)
	if err != nil {
		return err
	}
	if cpfExists {
		return fmt.Errorf("%w: cpf already in use", domain.ErrConflict)
	}

	return nil
}

func (s *ManagerService) mapManagers(ctx context.Context, managers []*domain.User, branchMap map[uuid.UUID]string) ([]SellerOutput, error) {
	output := make([]SellerOutput, 0, len(managers))
	for _, manager := range managers {
		item, err := s.mapManager(ctx, manager, branchMap)
		if err != nil {
			return nil, err
		}
		output = append(output, item)
	}
	return output, nil
}

func (s *ManagerService) mapManager(ctx context.Context, manager *domain.User, branchMap map[uuid.UUID]string) (SellerOutput, error) {
	item := SellerOutput{
		ID:        manager.ID,
		Name:      manager.Name,
		Email:     manager.Email,
		Phone:     manager.Phone,
		CPF:       manager.CPF,
		PhotoURL:  manager.PhotoURL,
		Role:      manager.Role,
		Status:    manager.Status,
		BranchID:  manager.BranchID,
		CreatedAt: manager.CreatedAt,
		UpdatedAt: manager.UpdatedAt,
	}

	if manager.BranchID != nil {
		if name, ok := branchMap[*manager.BranchID]; ok {
			item.BranchName = name
		} else {
			// Fallback for single object mapping if map is not provided
			branch, err := s.branchRepo.GetByID(ctx, *manager.BranchID)
			if err == nil && branch != nil {
				item.BranchName = branch.Name
			}
		}
	}

	if manager.BirthDate != nil {
		item.BirthDate = manager.BirthDate.Format("2006-01-02")
	}

	return item, nil
}
