package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleDirector    Role = "DIRECTOR"
	RoleManager     Role = "MANAGER"
	RoleSalesperson Role = "SALESPERSON"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusInactive UserStatus = "INACTIVE"
	UserStatusBlocked  UserStatus = "BLOCKED"
)

type User struct {
	ID                 uuid.UUID  `json:"id"`
	Name               string     `json:"name"`
	Email              string     `json:"email"`
	Phone              string     `json:"phone"`
	CPF                string     `json:"cpf"`
	PhotoURL           string     `json:"photoUrl"`
	BirthDate          *time.Time `json:"birthDate"`
	PasswordHash       string     `json:"-"`
	Role               Role       `json:"role"`
	Status             UserStatus `json:"status"`
	MustChangePassword bool       `json:"mustChangePassword"`
	BranchID           *uuid.UUID `json:"branchId"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
}

func (u *User) IsDirector() bool {
	return u != nil && u.Role == RoleDirector
}

func (u *User) IsManager() bool {
	return u != nil && u.Role == RoleManager
}

func (u *User) IsSalesperson() bool {
	return u != nil && u.Role == RoleSalesperson
}

func (u *User) CanManageSellers() bool {
	return u.IsDirector() || u.IsManager()
}

func (u *User) CanManageClients() bool {
	return u.IsDirector() || u.IsManager() || u.IsSalesperson()
}

func (u *User) CanViewClients() bool {
	return u.IsDirector() || u.IsManager() || u.IsSalesperson()
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id uuid.UUID) error
}
