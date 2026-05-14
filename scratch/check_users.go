package main

import (
	"fmt"
	"log"

	"lockcenter-backend/internal/config"
	"lockcenter-backend/internal/infrastructure/persistence"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	db, err := persistence.NewDatabase(cfg)
	if err != nil {
		log.Fatal(err)
	}

	var users []struct {
		Name               string
		Email              string
		CPF                *string
		Role               string
		Status             string
		MustChangePassword bool
		BranchID           *string
	}

	result := db.Gorm.Raw("SELECT name, email, cpf, role, status, must_change_password, branch_id FROM users").Scan(&users)
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	fmt.Println("Users in DB:")
	for _, u := range users {
		branchID := "NULL"
		if u.BranchID != nil {
			branchID = *u.BranchID
		}
		fmt.Printf("- Name: %s, Email: %s, Role: %s, Status: %s, MustChange: %v, BranchID: %s\n", u.Name, u.Email, u.Role, u.Status, u.MustChangePassword, branchID)
	}
}
