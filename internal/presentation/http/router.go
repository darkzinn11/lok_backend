package rest

import (
	"lockcenter-backend/internal/domain"
	"lockcenter-backend/internal/presentation/http/handlers"
	"lockcenter-backend/internal/presentation/http/middleware"

	"github.com/go-chi/chi/v5"
	mid "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(
	allowedOrigins []string,
	jwtManager middleware.TokenValidator,
	authHandler *handlers.AuthHandler,
	dashboardHandler *handlers.DashboardHandler,
	branchHandler *handlers.BranchHandler,
	sellerHandler *handlers.SellerHandler,
	managerHandler *handlers.ManagerHandler,
	visitHandler *handlers.VisitHandler,
	clientHandler *handlers.ClientHandler,
) *chi.Mux {
	r := chi.NewRouter()

	// Basic CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))

	r.Use(mid.RequestID)
	r.Use(mid.RealIP)
	r.Use(mid.Logger)
	r.Use(mid.Recoverer)
	r.Use(mid.RedirectSlashes)

	r.Route("/api/v1", func(r chi.Router) {
		// Public Auth Routes
		r.Group(func(r chi.Router) {
			r.Post("/auth/login", authHandler.Login)
			r.Post("/auth/refresh", authHandler.Refresh)
			r.Post("/auth/logout", authHandler.Logout)
		})

		// Protected Routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(jwtManager))

			r.Get("/auth/me", authHandler.Me)
			r.Patch("/auth/change-password", authHandler.ChangePassword)

			// Director Only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RBACMiddleware(domain.RoleDirector))
				r.Get("/managers", managerHandler.List)
				r.Post("/managers", managerHandler.Create)
				r.Get("/managers/{managerID}", managerHandler.GetByID)
				r.Put("/managers/{managerID}", managerHandler.Update)
				r.Delete("/managers/{managerID}", managerHandler.Delete)
			})

			// Dashboard access for Salesperson, Manager and Director
			r.Group(func(r chi.Router) {
				r.Use(middleware.RBACMiddleware(domain.RoleManager, domain.RoleSalesperson))
				r.Get("/dashboard/overview", dashboardHandler.Overview)
			})

			// Manager and Director Only
			r.Group(func(r chi.Router) {
				r.Use(middleware.RBACMiddleware(domain.RoleManager))
				r.Get("/branches", branchHandler.List)
				r.Get("/sellers", sellerHandler.List)
				r.Post("/sellers", sellerHandler.Create)
				r.Get("/sellers/{sellerID}", sellerHandler.GetByID)
				r.Put("/sellers/{sellerID}", sellerHandler.Update)
				r.Delete("/sellers/{sellerID}", sellerHandler.Delete)
				r.Get("/dashboard/reports/sellers", dashboardHandler.SellerReport)
			})

			r.Route("/visits", func(r chi.Router) {
				r.Get("/", visitHandler.List)
				r.Get("/kpis", visitHandler.SellerKPIs)
				r.Get("/{visitID}", visitHandler.GetByID)

				r.Group(func(r chi.Router) {
					r.Use(middleware.RBACMiddleware(domain.RoleSalesperson))
					r.Post("/", visitHandler.Create)
				})

				r.Group(func(r chi.Router) {
					r.Use(middleware.RBACMiddleware(domain.RoleManager))
					r.Put("/{visitID}", visitHandler.Update)
					r.Delete("/{visitID}", visitHandler.Delete)
				})
			})

			r.Route("/clients", func(r chi.Router) {
				r.Get("/", clientHandler.List)
				r.Get("/{id}", clientHandler.GetByID)
				r.Post("/", clientHandler.Create)
				r.Put("/{id}", clientHandler.Update)
				r.Delete("/{id}", clientHandler.Delete)

				r.Group(func(r chi.Router) {
					r.Use(middleware.RBACMiddleware(domain.RoleManager))
					r.Get("/stale", clientHandler.ListStale)
					r.Post("/{id}/reassign", clientHandler.Reassign)
				})
			})
		})
	})

	return r
}
