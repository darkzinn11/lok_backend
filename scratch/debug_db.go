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
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	db, err := persistence.NewDatabase(cfg)
	if err != nil {
		log.Fatalf("Erro ao conectar no banco: %v", err)
	}

	fmt.Println("--- USERS ---")
	var users []map[string]interface{}
	db.Gorm.Table("users").Find(&users)
	for _, u := range users {
		fmt.Printf("ID: %v | Name: %v | Email: %v | Role: %v | BranchID: %v\n",
			u["id"], u["name"], u["email"], u["role"], u["branch_id"])
	}

	fmt.Println("\n--- BRANCHES ---")
	var branches []map[string]interface{}
	db.Gorm.Table("branches").Find(&branches)
	for _, b := range branches {
		fmt.Printf("ID: %v | Name: %v\n", b["id"], b["name"])
	}

	fmt.Println("\n--- VISITS ---")
	var visits []map[string]interface{}
	db.Gorm.Table("visits").Find(&visits)
	for _, v := range visits {
		fmt.Printf("ID: %v | Date: %v | Status: %v | SalespersonID: %v | Client: %v\n",
			v["id"], v["date"], v["status"], v["salesperson_id"], v["client_name"])
	}
}
