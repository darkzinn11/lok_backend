package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lockcenter-backend/internal/application"
	"lockcenter-backend/internal/config"
	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/infrastructure/auth"
	"lockcenter-backend/internal/infrastructure/persistence"
	"lockcenter-backend/internal/infrastructure/security"
	"lockcenter-backend/internal/infrastructure/storage"
	rest "lockcenter-backend/internal/presentation/http"
	"lockcenter-backend/internal/presentation/http/handlers"
)

func main() {
	// 1. Configure Observability
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	// 2. Load Config
	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", slog.Any("error", err))
		os.Exit(1)
	}

	// 2.1. Ensure Directories exist
	if cfg.UploadDir != "" {
		if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
			slog.Error("Failed to create upload directory", slog.String("path", cfg.UploadDir), slog.Any("error", err))
		}
	}
	if cfg.LogDir != "" {
		if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
			slog.Error("Failed to create log directory", slog.String("path", cfg.LogDir), slog.Any("error", err))
		}
	}

	// 3. Database & Persistance Layer
	db, err := persistence.NewDatabase(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", slog.Any("error", err))
		os.Exit(1)
	}

	userRepo := persistence.NewGormUserRepository(db.Gorm)
	authRepo := persistence.NewGormAuthRepository(db.Gorm)
	branchRepo := persistence.NewGormBranchRepository(db.Gorm)
	sellerRepo := persistence.NewGormSellerRepository(db.Gorm)
	managerRepo := persistence.NewGormManagerRepository(db.Gorm)
	visitRepo := persistence.NewGormVisitRepository(db.Gorm)
	clientRepo := persistence.NewGormClientRepository(db.Gorm)

	// 4. Infrastructure & Security Layer
	tokenManager := auth.NewJWTTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	passwordHasher := security.NewBcryptHasher(12)

	// 4.1 Storage Provider
	var storageProvider domain.StorageProvider
	if cfg.AWSAccessKey != "" && cfg.AWSSecretKey != "" {
		minioProv, err := storage.NewMinioProvider(cfg.AWSEndpoint, cfg.AWSAccessKey, cfg.AWSSecretKey, cfg.AWSUseSSL)
		if err != nil {
			slog.Error("Failed to initialize Minio provider, falling back to local storage", slog.Any("error", err))
			storageProvider = storage.NewLocalStorage(cfg.UploadDir)
		} else {
			storageProvider = minioProv
		}
	} else {
		storageProvider = storage.NewLocalStorage(cfg.UploadDir)
	}

	// 5. Application Layer
	imageService := application.NewImageService(storageProvider, cfg.AWSBucket)
	authService := application.NewAuthService(userRepo, authRepo, tokenManager, passwordHasher)
	branchService := application.NewBranchService(branchRepo)
	sellerService := application.NewSellerService(sellerRepo, userRepo, branchRepo, passwordHasher)
	managerService := application.NewManagerService(managerRepo, userRepo, branchRepo, passwordHasher)
	visitService := application.NewVisitService(visitRepo, userRepo, branchRepo, clientRepo)
	dashboardService := application.NewDashboardService(visitRepo, userRepo, sellerRepo, branchRepo, clientRepo)
	clientService := application.NewClientService(clientRepo, userRepo, branchRepo)

	// 6. Presentation Layer (Handlers)
	isProd := cfg.AppEnv == "production"
	authHandler := handlers.NewAuthHandler(authService, isProd)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)
	branchHandler := handlers.NewBranchHandler(branchService)
	sellerHandler := handlers.NewSellerHandler(sellerService)
	managerHandler := handlers.NewManagerHandler(managerService)
	visitHandler := handlers.NewVisitHandler(visitService)
	clientHandler := handlers.NewClientHandler(clientService)
	uploadHandler := handlers.NewUploadHandler(imageService)

	// 7. Router Setup
	r := rest.NewRouter(cfg.CORSAllowedOrigins, tokenManager, authHandler, dashboardHandler, branchHandler, sellerHandler, managerHandler, visitHandler, clientHandler, uploadHandler)

	// Global Health Check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok", "environment":"%s", "time":"%s"}`, cfg.AppEnv, time.Now().Format(time.RFC3339))
	})

	// 8. Server Start / Graceful Shutdown
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("Server is starting", slog.String("address", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed ListenAndServe", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", slog.Any("error", err))
	}

	slog.Info("Server exited.")
}
