.PHONY: all build test run clean install dev backend frontend lint install-service uninstall-service restart-service service-logs

all: build

install:
	cd backend && go mod tidy
	cd frontend && npm ci

# Dev: starts both backend and frontend
dev:
	@trap 'kill 0' EXIT; \
	cd backend && go run ./cmd/cmux & \
	cd frontend && npm run dev & \
	wait

# Build
backend:
	cd backend && go build -o bin/cmux ./cmd/cmux
ifeq ($(shell uname -s),Darwin)
	codesign --force --options runtime --sign - backend/bin/cmux
endif

frontend:
	cd frontend && npm run build

build: frontend backend

# Test
test:
	cd backend && go test ./...
	cd frontend && npm run test:run

# Lint
lint:
	cd backend && golangci-lint run ./...
	cd frontend && npm run lint

clean:
	rm -rf backend/bin backend/db/cmux.db backend/internal/static/dist frontend/dist

# --- macOS Service (launchd) ---

PLIST_SRC    := com.corwind.cmux.plist
PLIST_DEST   := $(HOME)/Library/LaunchAgents/com.corwind.cmux.plist
SERVICE_TARGET := gui/$(shell id -u)/com.corwind.cmux
BIN_DEST     := $(HOME)/.local/bin/cmux
DATA_DIR     := $(HOME)/.cmux

install-service: backend
	@mkdir -p $(DATA_DIR)
	@mkdir -p $(dir $(BIN_DEST))
	cp backend/bin/cmux $(BIN_DEST)
	sed 's|__HOME__|$(HOME)|g' $(PLIST_SRC) > $(PLIST_DEST)
	-launchctl bootout $(SERVICE_TARGET) 2>/dev/null
	launchctl bootstrap gui/$(shell id -u) $(PLIST_DEST)
	@echo "cmux service installed and started"

uninstall-service:
	-launchctl bootout $(SERVICE_TARGET) 2>/dev/null
	-rm -f $(PLIST_DEST)
	-rm -f $(BIN_DEST)
	@echo "cmux service uninstalled"

restart-service: backend
	cp backend/bin/cmux $(BIN_DEST)
	-launchctl bootout $(SERVICE_TARGET) 2>/dev/null
	launchctl bootstrap gui/$(shell id -u) $(PLIST_DEST)
	@echo "cmux service restarted"

service-logs:
	tail -f $(DATA_DIR)/cmux.log
