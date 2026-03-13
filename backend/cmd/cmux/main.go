package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	appservice "github.com/Corwind/cmux/backend/internal/app"
	httpadapter "github.com/Corwind/cmux/backend/internal/adapters/http"
	"github.com/Corwind/cmux/backend/internal/adapters/filesystem"
	"github.com/Corwind/cmux/backend/internal/adapters/pty"
	"github.com/Corwind/cmux/backend/internal/adapters/pty/sandbox"
	"github.com/Corwind/cmux/backend/internal/adapters/sqlite"
)

func main() {
	dbPath := os.Getenv("CMUX_DB_PATH")
	if dbPath == "" {
		dbPath = "db/cmux.db"
	}
	port := os.Getenv("CMUX_PORT")
	if port == "" {
		port = "3001"
	}

	repo, err := sqlite.NewRepository(dbPath)
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}

	templateRepo := sqlite.NewTemplateRepository(repo.DB())
	templateService := appservice.NewTemplateService(templateRepo)

	// Seed templates from sandbox-profiles directory if none exist
	seedTemplates(templateService)

	templateDir := os.Getenv("CMUX_SANDBOX_TEMPLATE_DIR")
	if templateDir == "" {
		templateDir = "sandbox-profiles"
	}
	builder := sandbox.NewProfileBuilder(templateDir)
	managerOpts := []pty.Option{pty.WithSandbox(builder)}

	if tmplEnv := os.Getenv("CMUX_SANDBOX_TEMPLATES"); tmplEnv != "" {
		templates := strings.Split(tmplEnv, ",")
		managerOpts = append(managerOpts, pty.WithSandboxTemplates(templates...))
	}
	log.Printf("sandbox enabled (template dir: %s)", templateDir)

	processManager := pty.NewManager(managerOpts...)
	fileBrowser := filesystem.NewBrowser()
	sessionService := appservice.NewSessionService(repo, processManager, templateRepo)

	router := httpadapter.NewRouter(sessionService, templateService, fileBrowser)

	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown on SIGTERM/SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		log.Printf("cmux server starting on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Printf("shutting down...")

	// Stop accepting new connections, wait up to 5s for in-flight requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	// Kill all PTY processes
	processManager.KillAll()

	// Close database
	_ = repo.Close()

	log.Printf("cmux stopped")
}

func seedTemplates(svc *appservice.TemplateService) {
	ctx := context.Background()
	templates, err := svc.ListTemplates(ctx)
	if err != nil {
		log.Printf("failed to list templates for seeding: %v", err)
		return
	}
	if len(templates) > 0 {
		return
	}

	profileDir := os.Getenv("CMUX_SANDBOX_TEMPLATE_DIR")
	if profileDir == "" {
		profileDir = "sandbox-profiles"
	}

	entries, err := os.ReadDir(profileDir)
	if err != nil {
		log.Printf("no sandbox-profiles directory found, skipping template seeding")
		return
	}

	var allContent []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sbpl") {
			continue
		}
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", profileDir, entry.Name()))
		if err != nil {
			log.Printf("failed to read %s: %v", entry.Name(), err)
			continue
		}
		content := string(data)
		name := strings.TrimSuffix(entry.Name(), ".sbpl")

		if _, err := svc.CreateTemplate(ctx, name, content); err != nil {
			log.Printf("failed to seed template %s: %v", name, err)
			continue
		}
		allContent = append(allContent, content)
		log.Printf("seeded template: %s", name)
	}

	// Create a combined "Standard" template and set as default
	if len(allContent) > 0 {
		combined := strings.Join(allContent, "\n\n")
		tmpl, err := svc.CreateTemplate(ctx, "Standard", combined)
		if err != nil {
			log.Printf("failed to create Standard template: %v", err)
			return
		}
		if err := svc.SetDefault(ctx, tmpl.ID); err != nil {
			log.Printf("failed to set Standard as default: %v", err)
		} else {
			log.Printf("set Standard template as default")
		}
	}
}
