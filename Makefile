.PHONY: all build test run clean docker-up docker-down backend-test frontend-test backend-build frontend-build install

all: build

install:
	cd backend && go mod tidy
	cd frontend && npm ci

# Backend
backend-build:
	cd backend && go build -o bin/cmux ./cmd/cmux

backend-test:
	cd backend && go test ./...

backend-run:
	cd backend && go run ./cmd/cmux

backend-lint:
	cd backend && golangci-lint run ./...

# Frontend
frontend-build:
	cd frontend && npm run build

frontend-test:
	cd frontend && npm run test:run

frontend-dev:
	cd frontend && npm run dev

frontend-lint:
	cd frontend && npm run lint

# Combined
build: backend-build frontend-build

test: backend-test frontend-test

lint: backend-lint frontend-lint

run: backend-run

# Docker
docker-up:
	docker compose up --build

docker-down:
	docker compose down

clean:
	rm -rf backend/bin backend/db/cmux.db frontend/dist frontend/node_modules
