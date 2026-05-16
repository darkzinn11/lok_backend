package persistence

import (
	"context"
	"strings"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GormRepository provides a base for GORM repositories
type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// User Repository
type GormUserRepository struct {
	db *gorm.DB
}

func NewGormUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

func (r *GormUserRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *GormUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *GormUserRepository) List(ctx context.Context) ([]*domain.User, error) {
	var users []*domain.User
	if err := r.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

func (r *GormUserRepository) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *GormUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.User{}, "id = ?", id).Error
}

// Visit Repository
type GormVisitRepository struct {
	db *gorm.DB
}

func NewGormVisitRepository(db *gorm.DB) *GormVisitRepository {
	return &GormVisitRepository{db: db}
}

func (r *GormVisitRepository) Create(ctx context.Context, visit *domain.Visit) error {
	return r.db.WithContext(ctx).Create(visit).Error
}

func (r *GormVisitRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Visit, error) {
	var visit domain.Visit
	if err := r.db.WithContext(ctx).Preload("Photos").First(&visit, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &visit, nil
}

func (r *GormVisitRepository) List(ctx context.Context, filters domain.VisitFilters) ([]*domain.Visit, error) {
	var visits []*domain.Visit
	query := r.buildVisitQuery(ctx, filters)

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	// For general list, we might want photos, but for dashboard counts we use Count()
	// To optimize, we only preload if specifically needed or if it's a small list
	if filters.Limit <= 50 {
		query = query.Preload("Photos")
	}

	if err := query.Order("visits.date DESC, visits.created_at DESC").Find(&visits).Error; err != nil {
		return nil, err
	}

	return visits, nil
}

func (r *GormVisitRepository) Count(ctx context.Context, filters domain.VisitFilters) (int64, error) {
	var count int64
	query := r.buildVisitQuery(ctx, filters)
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

func (r *GormVisitRepository) buildVisitQuery(ctx context.Context, filters domain.VisitFilters) *gorm.DB {
	query := r.db.WithContext(ctx).Model(&domain.Visit{})

	if filters.SalespersonID != nil || filters.BranchID != nil || strings.TrimSpace(filters.Search) != "" {
		query = query.Joins("JOIN users ON users.id = visits.salesperson_id")
	}

	if filters.SalespersonID != nil {
		query = query.Where("visits.salesperson_id = ?", *filters.SalespersonID)
	}

	if filters.BranchID != nil {
		query = query.Where("users.branch_id = ?", *filters.BranchID)
	}

	if filters.Status != nil {
		query = query.Where("visits.status = ?", *filters.Status)
	}

	if filters.Date != nil {
		query = query.Where("visits.date = ?", filters.Date.Format("2006-01-02"))
	}

	if filters.StartDate != nil {
		query = query.Where("visits.date >= ?", filters.StartDate.Format("2006-01-02"))
	}

	if filters.EndDate != nil {
		query = query.Where("visits.date <= ?", filters.EndDate.Format("2006-01-02"))
	}

	if subject := strings.TrimSpace(filters.Subject); subject != "" {
		query = query.Where("LOWER(visits.subject) = ?", strings.ToLower(subject))
	}

	if conclusion := strings.TrimSpace(filters.Conclusion); conclusion != "" {
		query = query.Where("LOWER(visits.conclusion) = ?", strings.ToLower(conclusion))
	}

	if search := strings.TrimSpace(strings.ToLower(filters.Search)); search != "" {
		like := "%" + search + "%"
		query = query.Where(
			"LOWER(visits.client_name) LIKE ? OR LOWER(visits.client_cnpj) LIKE ? OR LOWER(visits.subject) LIKE ? OR LOWER(COALESCE(visits.observations, '')) LIKE ? OR LOWER(users.name) LIKE ?",
			like, like, like, like, like,
		)
	}

	if filters.OnlyAlerts {
		cutoff := time.Now().AddDate(0, 0, -30)
		query = query.Where("visits.date < ? AND visits.status <> ?", cutoff.Format("2006-01-02"), domain.StatusCompleted)
	}

	return query
}

func (r *GormVisitRepository) Update(ctx context.Context, visit *domain.Visit) error {
	return r.db.WithContext(ctx).Save(visit).Error
}

func (r *GormVisitRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.VisitStatus) error {
	return r.db.WithContext(ctx).Model(&domain.Visit{}).Where("id = ?", id).Update("status", status).Error
}

func (r *GormVisitRepository) AddPhoto(ctx context.Context, photo *domain.VisitPhoto) error {
	return r.db.WithContext(ctx).Create(photo).Error
}

func (r *GormVisitRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Visit{}, "id = ?", id).Error
}
