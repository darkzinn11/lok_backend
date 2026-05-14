package application

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
)

type ClientAddress struct {
	Street       string `json:"street"`
	Number       string `json:"number"`
	Complement   string `json:"complement,omitempty"`
	Neighborhood string `json:"neighborhood"`
	City         string `json:"city"`
	UF           string `json:"uf"`
}

type ClientOutput struct {
	ID            uuid.UUID     `json:"id"`
	BranchID      uuid.UUID     `json:"branchId"`
	SellerID      uuid.UUID     `json:"sellerId"`
	Name          string        `json:"name"`
	CNPJ          string        `json:"cnpj"`
	Email         string        `json:"email"`
	ContactPhone  string        `json:"contactPhone"`
	FixedPhone    string        `json:"fixedPhone"`
	Address       ClientAddress `json:"address"`
	CreatedAt     time.Time     `json:"createdAt"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

type CreateClientInput struct {
	Name          string        `json:"name"`
	CNPJ          string        `json:"cnpj"`
	Email         string        `json:"email"`
	ContactPhone  string        `json:"contactPhone"`
	FixedPhone    string        `json:"fixedPhone"`
	Address       ClientAddress `json:"address"`
}

type UpdateClientInput struct {
	Name          string        `json:"name"`
	Email         string        `json:"email"`
	ContactPhone  string        `json:"contactPhone"`
	FixedPhone    string        `json:"fixedPhone"`
	Address       ClientAddress `json:"address"`
}

type ClientService struct {
	clientRepo domain.ClientRepository
	userRepo   domain.UserRepository
	branchRepo domain.BranchRepository
}

func NewClientService(clientRepo domain.ClientRepository, userRepo domain.UserRepository, branchRepo domain.BranchRepository) *ClientService {
	return &ClientService{
		clientRepo: clientRepo,
		userRepo:   userRepo,
		branchRepo: branchRepo,
	}
}

func (s *ClientService) Create(ctx context.Context, actorID uuid.UUID, input CreateClientInput) (*ClientOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if !actor.CanManageClients() {
		return nil, domain.ErrForbidden
	}

	branchID, err := s.resolveClientBranch(ctx, actor)
	if err != nil {
		return nil, err
	}

	cnpj := normalizeDigits(input.CNPJ)
	if cnpj == "" {
		return nil, fmt.Errorf("%w: CNPJ is required", domain.ErrValidation)
	}

	existing, _ := s.clientRepo.GetByCNPJ(ctx, cnpj, branchID)
	if existing != nil {
		return nil, fmt.Errorf("%w: client already registered in this branch", domain.ErrValidation)
	}

	addressJSON, err := json.Marshal(input.Address)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid address", domain.ErrValidation)
	}

	client := &domain.Client{
		ID:            uuid.New(),
		BranchID:      branchID,
		SellerID:      actor.ID,
		Name:          strings.TrimSpace(input.Name),
		CNPJ:          cnpj,
		Email:         strings.TrimSpace(input.Email),
		ContactPhone:  normalizeDigits(input.ContactPhone),
		FixedPhone:    normalizeDigits(input.FixedPhone),
		Address:       string(addressJSON),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	if client.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}

	if err := s.clientRepo.Create(ctx, client); err != nil {
		return nil, err
	}

	return s.mapClient(client), nil
}

func (s *ClientService) List(ctx context.Context, actorID uuid.UUID, search string) ([]*ClientOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if !actor.CanViewClients() {
		return nil, domain.ErrForbidden
	}

	filters := domain.ClientFilters{
		Search: search,
	}

	if actor.IsSalesperson() {
		filters.SellerID = &actor.ID
	} else if actor.BranchID != nil {
		filters.BranchID = actor.BranchID
	}

	clients, err := s.clientRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	output := make([]*ClientOutput, 0, len(clients))
	for _, client := range clients {
		output = append(output, s.mapClient(client))
	}
	return output, nil
}

func (s *ClientService) ListStale(ctx context.Context, actorID uuid.UUID) ([]*domain.StaleClient, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if actor.IsSalesperson() {
		return nil, domain.ErrForbidden
	}

	var branchID *uuid.UUID
	if actor.IsManager() {
		branchID = actor.BranchID
	}

	return s.clientRepo.ListStale(ctx, branchID, 90)
}

func (s *ClientService) Reassign(ctx context.Context, actorID uuid.UUID, clientID uuid.UUID, newSellerID uuid.UUID, reason string) error {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return domain.ErrUnauthorized
	}

	if !actor.IsDirector() && !actor.IsManager() {
		return domain.ErrForbidden
	}

	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		return domain.ErrNotFound
	}

	newSeller, err := s.userRepo.GetByID(ctx, newSellerID)
	if err != nil {
		return domain.ErrNotFound
	}

	// Validate branch scope for managers
	if actor.IsManager() {
		if actor.BranchID == nil || client.BranchID != *actor.BranchID {
			return domain.ErrForbidden
		}
		if newSeller.BranchID == nil || *newSeller.BranchID != *actor.BranchID {
			return domain.ErrForbidden
		}
	}

	// Calculate inactivity days for history
	staleClients, err := s.clientRepo.ListStale(ctx, &client.BranchID, 0)
	var inactiveDays int
	for _, sc := range staleClients {
		if sc.ID == client.ID {
			inactiveDays = sc.DaysInactive
			break
		}
	}

	history := &domain.ClientReassignment{
		ID:                         uuid.New(),
		ClientID:                   client.ID,
		PreviousSellerID:           client.SellerID,
		NewSellerID:                newSellerID,
		ReassignedBy:               actorID,
		Reason:                     reason,
		InactiveDaysAtReassignment: inactiveDays,
		CreatedAt:                  time.Now().UTC(),
	}

	client.SellerID = newSellerID
	client.UpdatedAt = time.Now().UTC()

	return s.clientRepo.Reassign(ctx, client, history)
}

func (s *ClientService) GetByID(ctx context.Context, actorID, id uuid.UUID) (*ClientOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if !actor.CanViewClients() {
		return nil, domain.ErrForbidden
	}

	client, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	if err := s.ensureAccess(actor, client); err != nil {
		return nil, err
	}

	return s.mapClient(client), nil
}

func (s *ClientService) Update(ctx context.Context, actorID, id uuid.UUID, input UpdateClientInput) (*ClientOutput, error) {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	if !actor.CanManageClients() {
		return nil, domain.ErrForbidden
	}

	client, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	if err := s.ensureAccess(actor, client); err != nil {
		return nil, err
	}

	addressJSON, err := json.Marshal(input.Address)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid address", domain.ErrValidation)
	}

	client.Name = strings.TrimSpace(input.Name)
	client.Email = strings.TrimSpace(input.Email)
	client.ContactPhone = normalizeDigits(input.ContactPhone)
	client.FixedPhone = normalizeDigits(input.FixedPhone)
	client.Address = string(addressJSON)
	client.UpdatedAt = time.Now().UTC()

	if client.Name == "" {
		return nil, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}

	if err := s.clientRepo.Update(ctx, client); err != nil {
		return nil, err
	}

	return s.mapClient(client), nil
}

func (s *ClientService) Delete(ctx context.Context, actorID, id uuid.UUID) error {
	actor, err := s.userRepo.GetByID(ctx, actorID)
	if err != nil {
		return domain.ErrUnauthorized
	}

	if !actor.CanManageClients() {
		return domain.ErrForbidden
	}

	client, err := s.clientRepo.GetByID(ctx, id)
	if err != nil {
		return domain.ErrNotFound
	}

	if err := s.ensureAccess(actor, client); err != nil {
		return err
	}

	return s.clientRepo.Delete(ctx, id)
}

func (s *ClientService) ensureAccess(actor *domain.User, client *domain.Client) error {
	if actor.IsDirector() {
		return nil
	}

	if actor.IsSalesperson() && client.SellerID == actor.ID {
		return nil
	}

	if actor.IsManager() && actor.BranchID != nil && client.BranchID == *actor.BranchID {
		return nil
	}

	return domain.ErrForbidden
}

func (s *ClientService) resolveClientBranch(ctx context.Context, actor *domain.User) (uuid.UUID, error) {
	if actor.BranchID != nil {
		return *actor.BranchID, nil
	}

	if !actor.IsDirector() {
		return uuid.Nil, domain.ErrForbidden
	}

	branches, err := s.branchRepo.List(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	for _, branch := range branches {
		if branch != nil && branch.Status == domain.BranchStatusActive {
			return branch.ID, nil
		}
	}

	return uuid.Nil, fmt.Errorf("%w: no active branch available", domain.ErrValidation)
}

func (s *ClientService) mapClient(client *domain.Client) *ClientOutput {
	var address ClientAddress
	_ = json.Unmarshal([]byte(client.Address), &address)

	return &ClientOutput{
		ID:            client.ID,
		BranchID:      client.BranchID,
		SellerID:      client.SellerID,
		Name:          client.Name,
		CNPJ:          client.CNPJ,
		Email:         client.Email,
		ContactPhone:  client.ContactPhone,
		FixedPhone:    client.FixedPhone,
		Address:       address,
		CreatedAt:     client.CreatedAt,
		UpdatedAt:     client.UpdatedAt,
	}
}
