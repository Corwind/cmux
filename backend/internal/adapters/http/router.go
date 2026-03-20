package http

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/Corwind/cmux/backend/internal/app"
	"github.com/Corwind/cmux/backend/internal/ports"
	"github.com/Corwind/cmux/backend/internal/static"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(sessionService *app.SessionService, templateService *app.TemplateService, fileBrowser ports.FileBrowser) http.Handler {
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

	// Serve embedded frontend (SPA with index.html fallback)
	mountSPA(r)

	return r
}

// mountSPA serves the embedded frontend assets. For any path that doesn't match
// a static file, it falls back to index.html to support client-side routing.
func mountSPA(r chi.Router) {
	distFS, err := fs.Sub(static.Assets, "dist")
	if err != nil {
		panic("failed to create sub filesystem for embedded assets: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(distFS))

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		// Try to open the file from the embedded FS
		if f, err := distFS.Open(path); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fallback: serve index.html for SPA client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
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
