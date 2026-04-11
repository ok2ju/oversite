package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/ok2ju/oversite/backend/internal/auth"
	customMw "github.com/ok2ju/oversite/backend/internal/middleware"
)

// NewRouter creates and configures the main chi router with middleware
// and route definitions.
func NewRouter(health *HealthHandler, authH *AuthHandler, demoH *DemoHandler, faceitH *FaceitHandler, tickH *TickHandler, rosterH *RosterHandler, eventH *EventHandler, heatmapH *HeatmapHandler, sessions auth.SessionStore) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(customMw.StructuredLogger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://localhost", "http://localhost:3000", "http://localhost"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", health.Healthz)
	r.Get("/readyz", health.Readyz)

	r.Route("/api/v1", func(r chi.Router) {
		// Public auth routes — no session required
		r.Route("/auth", func(r chi.Router) {
			r.Get("/faceit", authH.HandleLogin)
			r.Get("/faceit/callback", authH.HandleCallback)
			r.Post("/logout", authH.HandleLogout)
			r.Get("/me", authH.HandleMe)
		})

		// Protected routes — require valid session
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(sessions))
			r.Get("/demos", demoH.HandleList)
			r.Post("/demos", demoH.HandleUpload)
			r.Get("/demos/{id}", demoH.HandleGet)
			r.Delete("/demos/{id}", demoH.HandleDelete)
			r.Post("/faceit/sync", faceitH.HandleSync)
			r.Get("/demos/{id}/ticks", tickH.HandleGetTicks)
			r.Get("/demos/{id}/rounds/{roundNumber}/players", rosterH.HandleGetPlayers)
			r.Get("/demos/{id}/events", eventH.HandleGetEvents)
			r.Post("/heatmaps/aggregate", heatmapH.HandleAggregate)
		})
	})

	return r
}
