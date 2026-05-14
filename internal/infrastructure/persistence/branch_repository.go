package persistence

import (
	"context"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormBranchRepository struct {
	db *gorm.DB
}

func NewGormBranchRepository(db *gorm.DB) *GormBranchRepository {
	return &GormBranchRepository{db: db}
}

func (r *GormBranchRepository) List(ctx context.Context) ([]*domain.Branch, error) {
	var branches []*domain.Branch
	if err := r.db.WithContext(ctx).Order("name ASC").Find(&branches).Error; err != nil {
		return nil, err
	}

	return branches, nil
}

func (r *GormBranchRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Branch, error) {
	var branch domain.Branch
	if err := r.db.WithContext(ctx).First(&branch, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &branch, nil
}

func (r *GormBranchRepository) Create(ctx context.Context, branch *domain.Branch) error {
	return r.db.WithContext(ctx).Create(branch).Error
}
