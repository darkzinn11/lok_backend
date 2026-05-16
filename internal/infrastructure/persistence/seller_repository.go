package persistence

import (
	"context"
	"strings"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormSellerRepository struct {
	db *gorm.DB
}

func NewGormSellerRepository(db *gorm.DB) *GormSellerRepository {
	return &GormSellerRepository{db: db}
}

func (r *GormSellerRepository) List(ctx context.Context, filters domain.SellerFilters) ([]*domain.User, error) {
	var sellers []*domain.User

	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ?", domain.RoleSalesperson)

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

	if err := query.Order("name ASC").Find(&sellers).Error; err != nil {
		return nil, err
	}

	return sellers, nil
}

func (r *GormSellerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var seller domain.User
	if err := r.db.WithContext(ctx).
		Where("id = ? AND role = ?", id, domain.RoleSalesperson).
		First(&seller).Error; err != nil {
		return nil, err
	}

	return &seller, nil
}

func (r *GormSellerRepository) Create(ctx context.Context, seller *domain.User) error {
	return r.db.WithContext(ctx).Create(seller).Error
}

func (r *GormSellerRepository) Update(ctx context.Context, seller *domain.User) error {
	return r.db.WithContext(ctx).Save(seller).Error
}

func (r *GormSellerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, "id = ? AND role = ?", id, domain.RoleSalesperson).Error
}

func (r *GormSellerRepository) ExistsByEmail(ctx context.Context, email string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ? AND LOWER(email) = LOWER(?)", domain.RoleSalesperson, email)

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GormSellerRepository) ExistsByPhone(ctx context.Context, phone string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ? AND phone = ?", domain.RoleSalesperson, phone)

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GormSellerRepository) ExistsByCPF(ctx context.Context, cpf string, excludeID *uuid.UUID) (bool, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ? AND cpf = ?", domain.RoleSalesperson, cpf)

	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *GormSellerRepository) Count(ctx context.Context, filters domain.SellerFilters) (int, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("role = ?", domain.RoleSalesperson)

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
