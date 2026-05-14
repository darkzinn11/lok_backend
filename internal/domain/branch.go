package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type BranchStatus string

const (
	BranchStatusActive   BranchStatus = "ACTIVE"
	BranchStatusInactive BranchStatus = "INACTIVE"
)

type Branch struct {
	ID        uuid.UUID
	Name      string
	City      string
	UF        string
	Status    BranchStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type BranchRepository interface {
	List(ctx context.Context) ([]*Branch, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Branch, error)
	Create(ctx context.Context, branch *Branch) error
}
