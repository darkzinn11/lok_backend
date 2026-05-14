package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	BranchID      uuid.UUID `gorm:"type:uuid;not null" json:"branchId"`
	SellerID      uuid.UUID `gorm:"type:uuid;not null" json:"sellerId"`
	Name          string    `gorm:"type:varchar(255);not null" json:"name"`
	CNPJ          string    `gorm:"type:varchar(20);not null" json:"cnpj"`
	Email         string    `gorm:"type:varchar(255)" json:"email"`
	ContactPhone  string    `gorm:"type:varchar(20)" json:"contactPhone"`
	FixedPhone    string    `gorm:"type:varchar(20)" json:"fixedPhone"`
	Address       string    `gorm:"type:jsonb;not null" json:"address"`
	CreatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
	UpdatedAt     time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"updatedAt"`
}

type ClientReassignment struct {
	ID                         uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	ClientID                   uuid.UUID `gorm:"type:uuid;not null" json:"clientId"`
	PreviousSellerID           uuid.UUID `gorm:"type:uuid;not null" json:"previousSellerId"`
	NewSellerID                uuid.UUID `gorm:"type:uuid;not null" json:"newSellerId"`
	ReassignedBy               uuid.UUID `gorm:"type:uuid;not null" json:"reassignedBy"`
	Reason                     string    `gorm:"type:text" json:"reason"`
	InactiveDaysAtReassignment int       `gorm:"type:integer" json:"inactiveDaysAtReassignment"`
	CreatedAt                  time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"createdAt"`
}

type StaleClient struct {
	Client
	LastVisitDate    *time.Time `json:"lastVisitDate"`
	DaysInactive     int        `json:"daysInactive"`
	CurrentSellerName string     `json:"currentSellerName"`
}

type ClientFilters struct {
	BranchID *uuid.UUID
	SellerID *uuid.UUID
	Search   string
}

type ClientRepository interface {
	Create(ctx context.Context, client *Client) error
	GetByID(ctx context.Context, id uuid.UUID) (*Client, error)
	GetByCNPJ(ctx context.Context, cnpj string, branchID uuid.UUID) (*Client, error)
	List(ctx context.Context, filters ClientFilters) ([]*Client, error)
	ListStale(ctx context.Context, branchID *uuid.UUID, thresholdDays int) ([]*StaleClient, error)
	Update(ctx context.Context, client *Client) error
	Reassign(ctx context.Context, client *Client, history *ClientReassignment) error
	Delete(ctx context.Context, id uuid.UUID) error
}
