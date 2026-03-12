package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

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
		port = "8080"
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
	log.Printf("cmux server starting on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
