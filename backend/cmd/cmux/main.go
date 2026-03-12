package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appservice "github.com/Corwind/cmux/backend/internal/app"
	httpadapter "github.com/Corwind/cmux/backend/internal/adapters/http"
	"github.com/Corwind/cmux/backend/internal/adapters/filesystem"
	"github.com/Corwind/cmux/backend/internal/adapters/pty"
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

	processManager := pty.NewManager()
	fileBrowser := filesystem.NewBrowser()
	sessionService := appservice.NewSessionService(repo, processManager)

	router := httpadapter.NewRouter(sessionService, fileBrowser)

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
