package persistence

import (
	"context"
	"strings"
	"time"

	"lockcenter-backend/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormClientRepository struct {
	db *gorm.DB
}

func NewGormClientRepository(db *gorm.DB) *GormClientRepository {
	return &GormClientRepository{db: db}
}

func (r *GormClientRepository) Create(ctx context.Context, client *domain.Client) error {
	return r.db.WithContext(ctx).Create(client).Error
}

func (r *GormClientRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Client, error) {
	var client domain.Client
	if err := r.db.WithContext(ctx).First(&client, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *GormClientRepository) GetByCNPJ(ctx context.Context, cnpj string, branchID uuid.UUID) (*domain.Client, error) {
	var client domain.Client
	if err := r.db.WithContext(ctx).Where("cnpj = ? AND branch_id = ?", cnpj, branchID).First(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *GormClientRepository) List(ctx context.Context, filters domain.ClientFilters) ([]*domain.Client, error) {
	var clients []*domain.Client
	query := r.db.WithContext(ctx).Model(&domain.Client{})

	if filters.BranchID != nil {
		query = query.Where("branch_id = ?", *filters.BranchID)
	}

	if filters.SellerID != nil {
		query = query.Where("seller_id = ?", *filters.SellerID)
	}

	if search := strings.TrimSpace(strings.ToLower(filters.Search)); search != "" {
		like := "%" + search + "%"
		query = query.Where("LOWER(name) LIKE ? OR cnpj LIKE ?", like, like)
	}

	if err := query.Order("name ASC").Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

func (r *GormClientRepository) ListStale(ctx context.Context, branchID *uuid.UUID, thresholdDays int) ([]*domain.StaleClient, error) {
	var results []struct {
		domain.Client
		LastVisitDate     *time.Time `gorm:"column:last_visit_date"`
		DaysInactive      int        `gorm:"column:days_inactive"`
		CurrentSellerName string     `gorm:"column:current_seller_name"`
	}

	// This query finds the latest visit for each client by matching CNPJ and the seller's branch
	query := r.db.WithContext(ctx).
		Table("clients c").
		Select(`
			c.*, 
			v.max_date as last_visit_date, 
			EXTRACT(DAY FROM (CURRENT_TIMESTAMP - COALESCE(v.max_date, c.created_at)))::int as days_inactive,
			u.name as current_seller_name
		`).
		Joins("JOIN users u ON u.id = c.seller_id").
		Joins(`
			LEFT JOIN (
				SELECT v.client_cnpj, s.branch_id, MAX(v.date) as max_date
				FROM visits v
				JOIN users s ON s.id = v.salesperson_id
				WHERE v.status <> 'Rascunho'
				GROUP BY v.client_cnpj, s.branch_id
			) v ON v.client_cnpj = c.cnpj AND v.branch_id = c.branch_id
		`)

	if branchID != nil {
		query = query.Where("c.branch_id = ?", *branchID)
	}

	query = query.Where(`
		(v.max_date IS NULL AND EXTRACT(DAY FROM (CURRENT_TIMESTAMP - c.created_at)) >= ?) 
		OR 
		(v.max_date IS NOT NULL AND EXTRACT(DAY FROM (CURRENT_TIMESTAMP - v.max_date)) >= ?)
	`, thresholdDays, thresholdDays)

	if err := query.Order("days_inactive DESC").Scan(&results).Error; err != nil {
		return nil, err
	}

	output := make([]*domain.StaleClient, 0, len(results))
	for _, res := range results {
		res.Client.Address = res.Address // Ensure address JSON is preserved
		output = append(output, &domain.StaleClient{
			Client:            res.Client,
			LastVisitDate:     res.LastVisitDate,
			DaysInactive:      res.DaysInactive,
			CurrentSellerName: res.CurrentSellerName,
		})
	}

	return output, nil
}

func (r *GormClientRepository) Reassign(ctx context.Context, client *domain.Client, history *domain.ClientReassignment) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update client seller
		if err := tx.Save(client).Error; err != nil {
			return err
		}

		// Create history record
		if err := tx.Create(history).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *GormClientRepository) Update(ctx context.Context, client *domain.Client) error {
	return r.db.WithContext(ctx).Save(client).Error
}

func (r *GormClientRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&domain.Client{}, "id = ?", id).Error
}
