package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	appservice "github.com/Corwind/cmux/backend/internal/app"
	configadapter "github.com/Corwind/cmux/backend/internal/adapters/config"
	"github.com/Corwind/cmux/backend/internal/adapters/filesystem"
	gitadapter "github.com/Corwind/cmux/backend/internal/adapters/git"
	httpadapter "github.com/Corwind/cmux/backend/internal/adapters/http"
	"github.com/Corwind/cmux/backend/internal/adapters/pty"
	"github.com/Corwind/cmux/backend/internal/adapters/pty/sandbox"
	"github.com/Corwind/cmux/backend/internal/adapters/sqlite"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	cfg, err := configadapter.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	repo, err := sqlite.NewRepository(cfg.Server.DBPath)
	if err != nil {
		slog.Error("failed to initialize database", "err", err)
		os.Exit(1)
	}

	templateRepo := sqlite.NewTemplateRepository(repo.DB())
	templateService := appservice.NewTemplateService(templateRepo)

	// Seed templates from sandbox-profiles directory if none exist
	seedTemplates(templateService, cfg.Sandbox.TemplateDir)

	builder := sandbox.NewProfileBuilder(cfg.Sandbox.TemplateDir)
	slog.Info("resolving shell environment")
	envCache := configadapter.NewEnvCache(func() []string {
		return configadapter.ResolveShellEnv(cfg)
	}, 5*time.Minute)
	managerOpts := []pty.Option{pty.WithSandbox(builder), pty.WithEnvResolver(envCache.Get)}

	if len(cfg.Sandbox.Templates) > 0 {
		managerOpts = append(managerOpts, pty.WithSandboxTemplates(cfg.Sandbox.Templates...))
	}
	slog.Info("sandbox enabled", "template_dir", cfg.Sandbox.TemplateDir)

	processManager := pty.NewManager(managerOpts...)
	fileBrowser := filesystem.NewBrowser()
	gitService := gitadapter.NewService()
	worktreeRepo := sqlite.NewWorktreeRepository(repo)
	sessionService := appservice.NewSessionService(repo, processManager, templateRepo,
		appservice.WithGitService(gitService, cfg.Git.WorktreesDir),
		appservice.WithWorktreeRepository(worktreeRepo),
	)

	router := httpadapter.NewRouter(sessionService, templateService, fileBrowser, gitService, cfg.Server.Port)

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Graceful shutdown on SIGTERM/SIGINT
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		slog.Info("cmux server starting", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down")

	// Stop accepting new connections, wait up to 5s for in-flight requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	// Kill all PTY processes
	processManager.KillAll()

	// Close database
	_ = repo.Close()

	slog.Info("cmux stopped")
}

func seedTemplates(svc *appservice.TemplateService, profileDir string) {
	ctx := context.Background()
	templates, err := svc.ListTemplates(ctx)
	if err != nil {
		slog.Error("failed to list templates for seeding", "err", err)
		return
	}
	if len(templates) > 0 {
		return
	}

	entries, err := os.ReadDir(profileDir)
	if err != nil {
		slog.Info("no sandbox-profiles directory found, skipping template seeding")
		return
	}

	var allContent []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sbpl") {
			continue
		}
		data, err := os.ReadFile(fmt.Sprintf("%s/%s", profileDir, entry.Name()))
		if err != nil {
			slog.Error("failed to read template file", "file", entry.Name(), "err", err)
			continue
		}
		content := string(data)
		name := strings.TrimSuffix(entry.Name(), ".sbpl")

		if _, err := svc.CreateTemplate(ctx, name, content); err != nil {
			slog.Error("failed to seed template", "name", name, "err", err)
			continue
		}
		allContent = append(allContent, content)
		slog.Info("seeded template", "name", name)
	}

	// Create a combined "Standard" template and set as default
	if len(allContent) > 0 {
		combined := strings.Join(allContent, "\n\n")
		tmpl, err := svc.CreateTemplate(ctx, "Standard", combined)
		if err != nil {
			slog.Error("failed to create Standard template", "err", err)
			return
		}
		if err := svc.SetDefault(ctx, tmpl.ID); err != nil {
			slog.Error("failed to set Standard as default", "err", err)
		} else {
			slog.Info("set Standard template as default")
		}
	}
}
