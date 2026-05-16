package persistence

import (
	"context"
	"strings"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormManagerRepository struct {
	db *gorm.DB
}

func NewGormManagerRepository(db *gorm.DB) *GormManagerRepository {
	return &GormManagerRepository{db: db}
}

func (r *GormManagerRepository) List(ctx context.Context, filters domain.SellerFilters) ([]*domain.User, error) {
	var managers []*domain.User

	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ?", domain.RoleManager)

	if filters.BranchID != nil {
		query = query.Where("branch_id = ?", *filters.BranchID)
	}

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if search := strings.TrimSpace(strings.ToLower(filters.Search)); search != "" {
		like := "%" + search + "%"
		query = query.Where("LOWER(name) LIKE ? OR LOWER(email) LIKE ? OR phone LIKE ? OR cpf LIKE ?", like, like, like, like)
	}

	if err := query.Order("name ASC").Find(&managers).Error; err != nil {
		return nil, err
	}

	return managers, nil
}

func (r *GormManagerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var manager domain.User
	if err := r.db.WithContext(ctx).
		Where("id = ? AND role = ?", id, domain.RoleManager).
		First(&manager).Error; err != nil {
		return nil, err
	}

	return &manager, nil
}

func (r *GormManagerRepository) Create(ctx context.Context, manager *domain.User) error {
	return r.db.WithContext(ctx).Create(manager).Error
}

func (r *GormManagerRepository) Update(ctx context.Context, manager *domain.User) error {
	return r.db.WithContext(ctx).Save(manager).Error
}

func (r *GormManagerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, "id = ? AND role = ?", id, domain.RoleManager).Error
}

func (r *GormManagerRepository) ExistsByEmail(ctx context.Context, email string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("LOWER(email) = LOWER(?)", email)

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GormManagerRepository) ExistsByPhone(ctx context.Context, phone string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ? AND phone = ?", domain.RoleManager, phone)

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GormManagerRepository) ExistsByCPF(ctx context.Context, cpf string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ? AND cpf = ?", domain.RoleManager, cpf)

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GormManagerRepository) Count(ctx context.Context, filters domain.SellerFilters) (int, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ?", domain.RoleManager)

	if filters.BranchID != nil {
		query = query.Where("branch_id = ?", *filters.BranchID)
	}

	if filters.Status != nil {
		query = query.Where("status = ?", *filters.Status)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}

	return int(count), nil
}
