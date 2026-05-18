package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"lockcenter-backend/internal/application"
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

	// Active Gorm logger to see the generated SQL queries!
	db.Gorm = db.Gorm.Debug()

	// Instantiate repositories
	visitRepo := persistence.NewGormVisitRepository(db.Gorm)
	userRepo := persistence.NewGormUserRepository(db.Gorm)
	sellerRepo := persistence.NewGormSellerRepository(db.Gorm)
	branchRepo := persistence.NewGormBranchRepository(db.Gorm)
	clientRepo := persistence.NewGormClientRepository(db.Gorm)

	dashboardService := application.NewDashboardService(
		visitRepo,
		userRepo,
		sellerRepo,
		branchRepo,
		clientRepo,
	)

	// Fetch the Director user from DB
	var director domain.User
	if err := db.Gorm.First(&director, "role = ?", domain.RoleDirector).Error; err != nil {
		log.Fatalf("Erro ao buscar diretor: %v", err)
	}
	fmt.Printf("Director User: ID=%s, Name=%s\n", director.ID, director.Name)

	ctx := context.Background()

	// Define period: start_date=2026-01-01, end_date=2026-12-31
	startDate, _ := time.Parse("2006-01-02", "2026-01-01")
	endDate, _ := time.Parse("2006-01-02", "2026-12-31")

	fmt.Println("\n--- SCENARIO 1: ALL BRANCHES (BranchID = nil) ---")
	overviewAll, err := dashboardService.GetOverview(ctx, director.ID, application.DashboardRange{
		StartDate: startDate,
		EndDate:   endDate,
		BranchID:  nil,
	})
	if err != nil {
		log.Fatalf("Erro: %v", err)
	}
	fmt.Printf("ALL BRANCHES counts -> Total: %d, Pending: %d, Completed: %d\n",
		overviewAll.TotalVisits, overviewAll.PendingVisits, overviewAll.CompletedVisits)

	// Fetch Belém Branch
	var belemBranch domain.Branch
	if err := db.Gorm.First(&belemBranch, "name LIKE ?", "%Belém%").Error; err == nil {
		fmt.Printf("\nBelém Branch: ID=%s, Name=%s\n", belemBranch.ID, belemBranch.Name)

		fmt.Println("\n--- SCENARIO 2: BELÉM BRANCH ---")
		overviewBelem, err := dashboardService.GetOverview(ctx, director.ID, application.DashboardRange{
			StartDate: startDate,
			EndDate:   endDate,
			BranchID:  &belemBranch.ID,
		})
		if err != nil {
			log.Fatalf("Erro: %v", err)
		}
		fmt.Printf("BELÉM counts -> Total: %d, Pending: %d, Completed: %d\n",
			overviewBelem.TotalVisits, overviewBelem.PendingVisits, overviewBelem.CompletedVisits)
	} else {
		fmt.Println("Belém branch not found.")
	}

	// Fetch São Luís Sede Branch
	var slsBranch domain.Branch
	if err := db.Gorm.First(&slsBranch, "name LIKE ?", "%São Luís%").Error; err == nil {
		fmt.Printf("\nSão Luís Branch: ID=%s, Name=%s\n", slsBranch.ID, slsBranch.Name)

		fmt.Println("\n--- SCENARIO 3: SÃO LUÍS SE DE BRANCH ---")
		overviewSls, err := dashboardService.GetOverview(ctx, director.ID, application.DashboardRange{
			StartDate: startDate,
			EndDate:   endDate,
			BranchID:  &slsBranch.ID,
		})
		if err != nil {
			log.Fatalf("Erro: %v", err)
		}
		fmt.Printf("SÃO LUÍS counts -> Total: %d, Pending: %d, Completed: %d\n",
			overviewSls.TotalVisits, overviewSls.PendingVisits, overviewSls.CompletedVisits)
	}
}
