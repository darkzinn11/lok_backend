package application

import (
	"context"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
)

var operationalBranchID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

type BranchService struct {
	branchRepo domain.BranchRepository
}

type BranchOutput struct {
	ID        string              `json:"id"`
	Name      string              `json:"name"`
	City      string              `json:"city"`
	UF        string              `json:"uf"`
	Status    domain.BranchStatus `json:"status"`
	CreatedAt string              `json:"created_at"`
	UpdatedAt string              `json:"updated_at"`
}

func NewBranchService(branchRepo domain.BranchRepository) *BranchService {
	return &BranchService{branchRepo: branchRepo}
}

func (s *BranchService) List(ctx context.Context) ([]BranchOutput, error) {
	branches, err := s.branchRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	output := make([]BranchOutput, 0, len(branches))
	for _, branch := range branches {
		if branch.ID != operationalBranchID {
			continue
		}

		if branch.Status != domain.BranchStatusActive {
			continue
		}

		output = append(output, BranchOutput{
			ID:        branch.ID.String(),
			Name:      branch.Name,
			City:      branch.City,
			UF:        branch.UF,
			Status:    branch.Status,
			CreatedAt: branch.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: branch.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return output, nil
}
