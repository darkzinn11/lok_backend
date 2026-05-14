package domain

import (
	"context"

	"github.com/google/uuid"
)

type SellerFilters struct {
	Search   string
	BranchID *uuid.UUID
	Status   *UserStatus
}

type SellerRepository interface {
	List(ctx context.Context, filters SellerFilters) ([]*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	Create(ctx context.Context, seller *User) error
	Update(ctx context.Context, seller *User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ExistsByEmail(ctx context.Context, email string, excludeID *uuid.UUID) (bool, error)
	ExistsByPhone(ctx context.Context, phone string, excludeID *uuid.UUID) (bool, error)
	ExistsByCPF(ctx context.Context, cpf string, excludeID *uuid.UUID) (bool, error)
}
