package main

import (
	"fmt"
	"log"
	"time"

	"lockcenter-backend/internal/config"
	"lockcenter-backend/internal/domain"
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

	var visits []domain.Visit
	if err := db.Gorm.Find(&visits).Error; err != nil {
		log.Fatalf("Erro ao listar visitas: %v", err)
	}

	fmt.Printf("Total de visitas cadastradas no banco: %d\n", len(visits))
	for _, v := range visits {
		var user domain.User
		var branchName string = "N/A"
		if err := db.Gorm.First(&user, "id = ?", v.SalespersonID).Error; err == nil {
			if user.BranchID != nil {
				var branch domain.Branch
				if err := db.Gorm.First(&branch, "id = ?", *user.BranchID).Error; err == nil {
					branchName = branch.Name
				}
			}
		}
		fmt.Printf("- Visita ID: %s | Cliente: %s | Data: %s | Vendedor: %s | Filial do Vendedor: %s | Status: %s | CreatedAt: %s\n",
			v.ID, v.ClientName, v.Date.Format("2006-01-02"), user.Name, branchName, v.Status, v.CreatedAt.Format(time.RFC3339))
	}
}
