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
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: true,
	}))

	sessionHandler := NewSessionHandler(sessionService)
	fsHandler := NewFilesystemHandler(fileBrowser)
	wsHandler := NewWebSocketHandler(sessionService)

	r.Route("/api", func(r chi.Router) {
		r.Get("/sessions", sessionHandler.List)
		r.Post("/sessions", sessionHandler.Create)
		r.Get("/sessions/{id}", sessionHandler.Get)
		r.Delete("/sessions/{id}", sessionHandler.Delete)
		r.Get("/fs", fsHandler.ListDirectory)
	})

	r.Get("/ws/sessions/{id}", wsHandler.Handle)

	return r
}
