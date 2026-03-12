APP_NAME := $(shell node -p "require('./package.json').name")

.PHONY: install dev build preview lint format format-check typecheck test test-run test-e2e docker-dev docker-build docker-run clean

install:
	npm ci

dev:
	npm run dev

build:
	npm run build

preview:
	npm run preview

lint:
	npm run lint

format:
	npm run format

format-check:
	npm run format:check

typecheck:
	npm run typecheck

test:
	npm run test

test-run:
	npm run test:run

test-e2e:
	npx playwright test

docker-dev:
	docker compose up --build

docker-build:
	docker build -t $(APP_NAME) .

docker-run:
	docker run -p 8080:80 $(APP_NAME)

clean:
	rm -rf node_modules dist coverage playwright-report test-results
