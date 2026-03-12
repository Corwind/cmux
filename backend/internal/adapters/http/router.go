package http

import (
	"net/http"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/ports"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(sessionService *app.SessionService, fileBrowser ports.FileBrowser) http.Handler {
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
	fsHandler := NewFilesystemHandler(fileBrowser)
	wsHandler := NewWebSocketHandler(sessionService, WithOriginPatterns([]string{"localhost:5173", "localhost:3001"}))

	r.Route("/api", func(r chi.Router) {
		r.Get("/sessions", sessionHandler.List)
		r.Post("/sessions", sessionHandler.Create)
		r.Get("/sessions/{id}", sessionHandler.Get)
		r.Post("/sessions/{id}/resume", sessionHandler.Resume)
		r.Delete("/sessions/{id}", sessionHandler.Delete)
		r.Get("/fs", fsHandler.ListDirectory)
	})

	r.Get("/ws/sessions/{id}", wsHandler.Handle)

	return r
}

// NewTestRouter creates a router with permissive WebSocket origin patterns for testing.
func NewTestRouter(sessionService *app.SessionService, fileBrowser ports.FileBrowser) http.Handler {
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
	fsHandler := NewFilesystemHandler(fileBrowser)
	wsHandler := NewWebSocketHandler(sessionService, WithOriginPatterns([]string{"*"}))

	r.Route("/api", func(r chi.Router) {
		r.Get("/sessions", sessionHandler.List)
		r.Post("/sessions", sessionHandler.Create)
		r.Get("/sessions/{id}", sessionHandler.Get)
		r.Post("/sessions/{id}/resume", sessionHandler.Resume)
		r.Delete("/sessions/{id}", sessionHandler.Delete)
		r.Get("/fs", fsHandler.ListDirectory)
	})

	r.Get("/ws/sessions/{id}", wsHandler.Handle)

	return r
}
