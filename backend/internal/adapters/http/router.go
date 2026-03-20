package http

import (
	"io/fs"
	"net/http"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/ports"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(sessionService *app.SessionService, templateService *app.TemplateService, fileBrowser ports.FileBrowser, assets fs.FS) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3001"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	sessionHandler := NewSessionHandler(sessionService)
	templateHandler := NewTemplateHandler(templateService)
	fsHandler := NewFilesystemHandler(fileBrowser)
	wsHandler := NewWebSocketHandler(sessionService, WithOriginPatterns([]string{"localhost:5173", "localhost:3001"}))

	r.Route("/api", func(r chi.Router) {
		r.Get("/sessions", sessionHandler.List)
		r.Post("/sessions", sessionHandler.Create)
		r.Get("/sessions/{id}", sessionHandler.Get)
		r.Post("/sessions/{id}/resume", sessionHandler.Resume)
		r.Post("/sessions/{id}/restart", sessionHandler.Restart)
		r.Delete("/sessions/{id}", sessionHandler.Delete)

		r.Get("/templates", templateHandler.List)
		r.Post("/templates", templateHandler.Create)
		r.Post("/templates/import", templateHandler.Import)
		r.Delete("/templates/default", templateHandler.ClearDefault)
		r.Get("/templates/{id}", templateHandler.Get)
		r.Put("/templates/{id}", templateHandler.Update)
		r.Delete("/templates/{id}", templateHandler.Delete)
		r.Post("/templates/{id}/default", templateHandler.SetDefault)
		r.Get("/templates/{id}/export", templateHandler.Export)

		r.Get("/fs", fsHandler.ListDirectory)
	})

	r.Get("/ws/sessions/{id}", wsHandler.Handle)

	// Serve embedded frontend assets as catch-all (SPA)
	if assets != nil {
		r.Handle("/*", SPAHandler(assets))
	}

	return r
}

// NewTestRouter creates a router with permissive WebSocket origin patterns for testing.
func NewTestRouter(sessionService *app.SessionService, templateService *app.TemplateService, fileBrowser ports.FileBrowser) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3001"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	sessionHandler := NewSessionHandler(sessionService)
	templateHandler := NewTemplateHandler(templateService)
	fsHandler := NewFilesystemHandler(fileBrowser)
	wsHandler := NewWebSocketHandler(sessionService, WithOriginPatterns([]string{"*"}))

	r.Route("/api", func(r chi.Router) {
		r.Get("/sessions", sessionHandler.List)
		r.Post("/sessions", sessionHandler.Create)
		r.Get("/sessions/{id}", sessionHandler.Get)
		r.Post("/sessions/{id}/resume", sessionHandler.Resume)
		r.Post("/sessions/{id}/restart", sessionHandler.Restart)
		r.Delete("/sessions/{id}", sessionHandler.Delete)

		r.Get("/templates", templateHandler.List)
		r.Post("/templates", templateHandler.Create)
		r.Post("/templates/import", templateHandler.Import)
		r.Delete("/templates/default", templateHandler.ClearDefault)
		r.Get("/templates/{id}", templateHandler.Get)
		r.Put("/templates/{id}", templateHandler.Update)
		r.Delete("/templates/{id}", templateHandler.Delete)
		r.Post("/templates/{id}/default", templateHandler.SetDefault)
		r.Get("/templates/{id}/export", templateHandler.Export)

		r.Get("/fs", fsHandler.ListDirectory)
	})

	r.Get("/ws/sessions/{id}", wsHandler.Handle)

	return r
}
