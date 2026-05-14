package main

import (
	"fmt"
	"log"
	"time"

	"lockcenter-backend/internal/config"
	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/infrastructure/persistence"
	"lockcenter-backend/internal/infrastructure/security"

	"github.com/google/uuid"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	db, err := persistence.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Erro ao conectar no banco: %v", err)
	}

	hasher := security.NewBcryptHasher(12)

	// 1. Garantir existência de uma filial padrão
	branchID := uuid.New()
	branch := &domain.Branch{
		ID:        branchID,
		Name:      "Lokcenter Matriz",
		City:      "São Luís",
		UF:        "MA",
		Status:    "ACTIVE",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var existingBranch domain.Branch
	if err := db.Gorm.Where("name = ?", branch.Name).First(&existingBranch).Error; err != nil {
		fmt.Printf("Criando filial: %s...\n", branch.Name)
		if err := db.Gorm.Create(branch).Error; err != nil {
			log.Fatalf("Erro ao criar filial: %v", err)
		}
	} else {
		branchID = existingBranch.ID
		fmt.Printf("Filial '%s' já existe. ID: %s\n", existingBranch.Name, branchID)
	}

	// 2. Definir usuários para o Seed
	usersToCreate := []struct {
		Name     string
		Email    string
		Password string
		Role     domain.Role
		BranchID *uuid.UUID
	}{
		{
			Name:     "Diretor Lokcenter",
			Email:    "diretor@lokcenter.com.br",
			Password: "TrocarSenha123!",
			Role:     domain.RoleDirector,
			BranchID: nil,
		},
		{
			Name:     "Gerente Lokcenter",
			Email:    "gerente@lokcenter.com.br",
			Password: "TrocarSenha123!",
			Role:     domain.RoleManager,
			BranchID: &branchID,
		},
		{
			Name:     "Vendedor Lokcenter",
			Email:    "vendedor@lokcenter.com.br",
			Password: "TrocarSenha123!",
			Role:     domain.RoleSalesperson,
			BranchID: &branchID,
		},
	}

	fmt.Println("\nIniciando seed de usuários...")

	for _, u := range usersToCreate {
		var existingUser domain.User
		if err := db.Gorm.Where("email = ?", u.Email).First(&existingUser).Error; err == nil {
			fmt.Printf("✅ Usuário [%s] já existe: %s\n", u.Role, u.Email)
			continue
		}

		hash, err := hasher.Hash(u.Password)
		if err != nil {
			log.Fatalf("Erro ao gerar hash para %s: %v", u.Email, err)
		}

		user := &domain.User{
			ID:                 uuid.New(),
			Name:               u.Name,
			Email:              u.Email,
			PasswordHash:       hash,
			Role:               u.Role,
			Status:             domain.UserStatusActive,
			MustChangePassword: true, // Força troca no primeiro login
			BranchID:           u.BranchID,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		}

		if err := db.Gorm.Create(user).Error; err != nil {
			log.Fatalf("Erro ao criar usuário %s: %v", u.Email, err)
		}
		fmt.Printf("🚀 Usuário [%s] criado com sucesso: %s (Senha temporária: %s)\n", u.Role, u.Email, u.Password)
	}

	fmt.Println("\n✅ Seed finalizado com sucesso!")
}
