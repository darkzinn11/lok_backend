package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
)

func TestSellerServiceListScopesManagerToOwnBranch(t *testing.T) {
	t.Parallel()

	branchID := uuid.New()
	managerID := uuid.New()
	otherBranchID := uuid.New()

	userRepo := &stubUserRepository{
		usersByID: map[uuid.UUID]*domain.User{
			managerID: {
				ID:       managerID,
				Role:     domain.RoleManager,
				Status:   domain.UserStatusActive,
				BranchID: &branchID,
			},
		},
	}

	sellerRepo := &stubSellerRepository{
		listFn: func(_ context.Context, filters domain.SellerFilters) ([]*domain.User, error) {
			if filters.BranchID == nil || *filters.BranchID != branchID {
				t.Fatalf("expected manager scope to force branch %s, got %+v", branchID, filters.BranchID)
			}

			return []*domain.User{
				{
					ID:        uuid.New(),
					Name:      "Vendedor 1",
					Email:     "v1@lokcenter.com",
					Phone:     "98999999999",
					Role:      domain.RoleSalesperson,
					Status:    domain.UserStatusActive,
					BranchID:  &branchID,
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				},
			}, nil
		},
	}

	branchRepo := &stubBranchRepository{
		branchesByID: map[uuid.UUID]*domain.Branch{
			branchID:      {ID: branchID, Name: "Sede São Luís", Status: domain.BranchStatusActive},
			otherBranchID: {ID: otherBranchID, Name: "Filial Caxias", Status: domain.BranchStatusActive},
		},
	}

	service := NewSellerService(sellerRepo, userRepo, branchRepo, stubPasswordHasher{})

	result, err := service.List(context.Background(), managerID, domain.SellerFilters{
		BranchID: &otherBranchID,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 seller, got %d", len(result))
	}
	if result[0].BranchID == nil || *result[0].BranchID != branchID {
		t.Fatalf("expected seller output branch %s, got %+v", branchID, result[0].BranchID)
	}
	if result[0].PhotoURL != "" {
		t.Fatalf("expected empty photo url by default, got %q", result[0].PhotoURL)
	}
}

func TestSellerServiceCreateRejectsDuplicateEmail(t *testing.T) {
	t.Parallel()

	branchID := uuid.New()
	directorID := uuid.New()

	userRepo := &stubUserRepository{
		usersByID: map[uuid.UUID]*domain.User{
			directorID: {
				ID:     directorID,
				Role:   domain.RoleDirector,
				Status: domain.UserStatusActive,
			},
		},
	}

	sellerRepo := &stubSellerRepository{
		existsByEmail: true,
	}

	branchRepo := &stubBranchRepository{
		branchesByID: map[uuid.UUID]*domain.Branch{
			branchID: {ID: branchID, Name: "Sede São Luís", Status: domain.BranchStatusActive},
		},
	}

	service := NewSellerService(sellerRepo, userRepo, branchRepo, stubPasswordHasher{})

	_, err := service.Create(context.Background(), directorID, CreateSellerInput{
		Name:      "Maria Vendedora",
		Email:     "maria@lokcenter.com",
		Phone:     "(98) 99999-0000",
		CPF:       "123.456.789-09",
		PhotoURL:  "data:image/png;base64,abc",
		BirthDate: "1990-05-10",
		BranchID:  branchID,
		Status:    domain.UserStatusActive,
	})
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestSellerServiceUpdateRejectsManagerMovingSellerAcrossBranches(t *testing.T) {
	t.Parallel()

	managerBranchID := uuid.New()
	targetBranchID := uuid.New()
	managerID := uuid.New()
	sellerID := uuid.New()

	userRepo := &stubUserRepository{
		usersByID: map[uuid.UUID]*domain.User{
			managerID: {
				ID:       managerID,
				Role:     domain.RoleManager,
				Status:   domain.UserStatusActive,
				BranchID: &managerBranchID,
			},
		},
	}

	sellerRepo := &stubSellerRepository{
		sellersByID: map[uuid.UUID]*domain.User{
			sellerID: {
				ID:        sellerID,
				Name:      "Carlos",
				Email:     "carlos@lokcenter.com",
				Phone:     "98999998888",
				Role:      domain.RoleSalesperson,
				Status:    domain.UserStatusActive,
				BranchID:  &managerBranchID,
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			},
		},
	}

	branchRepo := &stubBranchRepository{
		branchesByID: map[uuid.UUID]*domain.Branch{
			managerBranchID: {ID: managerBranchID, Name: "Sede São Luís", Status: domain.BranchStatusActive},
			targetBranchID:  {ID: targetBranchID, Name: "Filial Imperatriz", Status: domain.BranchStatusActive},
		},
	}

	service := NewSellerService(sellerRepo, userRepo, branchRepo, stubPasswordHasher{})

	result, err := service.Update(context.Background(), managerID, sellerID, UpdateSellerInput{
		Name:      "Carlos Atualizado",
		Email:     "carlos@lokcenter.com",
		Phone:     "98999998888",
		CPF:       "12345678909",
		BirthDate: "1992-07-15",
		BranchID:  targetBranchID,
		Status:    domain.UserStatusInactive,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.BranchID == nil || *result.BranchID != managerBranchID {
		t.Fatalf("expected manager scope to keep seller in branch %s, got %+v", managerBranchID, result.BranchID)
	}
}

type stubSellerRepository struct {
	sellersByID   map[uuid.UUID]*domain.User
	listFn        func(context.Context, domain.SellerFilters) ([]*domain.User, error)
	existsByEmail bool
	existsByPhone bool
	existsByCPF   bool
	createdSeller *domain.User
	updatedSeller *domain.User
	deletedSeller uuid.UUID
}

func (r *stubSellerRepository) List(ctx context.Context, filters domain.SellerFilters) ([]*domain.User, error) {
	if r.listFn != nil {
		return r.listFn(ctx, filters)
	}
	return nil, nil
}

func (r *stubSellerRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	seller, ok := r.sellersByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return seller, nil
}

func (r *stubSellerRepository) Create(_ context.Context, seller *domain.User) error {
	r.createdSeller = seller
	return nil
}

func (r *stubSellerRepository) Update(_ context.Context, seller *domain.User) error {
	r.updatedSeller = seller
	return nil
}

func (r *stubSellerRepository) Delete(_ context.Context, id uuid.UUID) error {
	r.deletedSeller = id
	return nil
}

func (r *stubSellerRepository) ExistsByEmail(_ context.Context, _ string, _ *uuid.UUID) (bool, error) {
	return r.existsByEmail, nil
}

func (r *stubSellerRepository) ExistsByPhone(_ context.Context, _ string, _ *uuid.UUID) (bool, error) {
	return r.existsByPhone, nil
}

func (r *stubSellerRepository) ExistsByCPF(_ context.Context, _ string, _ *uuid.UUID) (bool, error) {
	return r.existsByCPF, nil
}

type stubUserRepository struct {
	usersByID map[uuid.UUID]*domain.User
}

func (r *stubUserRepository) Create(_ context.Context, _ *domain.User) error {
	return nil
}

func (r *stubUserRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	user, ok := r.usersByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return user, nil
}

func (r *stubUserRepository) GetByEmail(_ context.Context, _ string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (r *stubUserRepository) List(_ context.Context) ([]*domain.User, error) {
	return nil, nil
}

func (r *stubUserRepository) Update(_ context.Context, _ *domain.User) error {
	return nil
}

func (r *stubUserRepository) Delete(_ context.Context, _ uuid.UUID) error {
	return nil
}

type stubBranchRepository struct {
	branchesByID map[uuid.UUID]*domain.Branch
}

func (r *stubBranchRepository) List(_ context.Context) ([]*domain.Branch, error) {
	branches := make([]*domain.Branch, 0, len(r.branchesByID))
	for _, branch := range r.branchesByID {
		branches = append(branches, branch)
	}
	return branches, nil
}

func (r *stubBranchRepository) GetByID(_ context.Context, id uuid.UUID) (*domain.Branch, error) {
	branch, ok := r.branchesByID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return branch, nil
}

func (r *stubBranchRepository) Create(_ context.Context, branch *domain.Branch) error {
	if r.branchesByID == nil {
		r.branchesByID = make(map[uuid.UUID]*domain.Branch)
	}
	r.branchesByID[branch.ID] = branch
	return nil
}

type stubPasswordHasher struct{}

func (stubPasswordHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (stubPasswordHasher) Compare(hash, password string) error {
	if hash != "hashed:"+password {
		return errors.New("invalid password")
	}
	return nil
}
