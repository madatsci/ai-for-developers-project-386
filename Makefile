.DEFAULT_GOAL := help

-include .env

export DATABASE_URL ?= postgres://calendar:calendar@127.0.0.1:5433/calendar?sslmode=disable
export HTTP_ADDR ?= 127.0.0.1:8080
export API_PROXY_TARGET ?= http://127.0.0.1:8080
export FRONTEND_PORT ?= 5175
SMOKE_APP_PORT ?= 18080
SMOKE_POSTGRES_PORT ?= 15433
SMOKE_PROJECT ?= calendar-smoke
STANDALONE_SMOKE_APP_PORT ?= 18081
STANDALONE_SMOKE_CONTAINER ?= calendar-standalone-smoke
STANDALONE_SMOKE_IMAGE ?= calendar-of-calls:standalone-smoke

GOLANGCI_VERSION := v2.15.0
GOLANGCI := $(CURDIR)/.bin/$(GOLANGCI_VERSION)/golangci-lint

.PHONY: help setup install tools browser db-up db-down migrate seed dev dev-backend dev-frontend test test-backend test-integration test-frontend lint lint-backend lint-frontend lint-typespec lint-openapi lint-client lint-server lint-render format build build-backend build-frontend build-typespec generate generate-client generate-server container-build smoke smoke-standalone smoke-remote check

help:
	@echo "make setup        Install dependencies, Chromium and the pinned Go linter"
	@echo "make dev          Start PostgreSQL, backend and frontend"
	@echo "make migrate      Apply PostgreSQL schema changes"
	@echo "make seed         Create the initial organizer configuration"
	@echo "make test         Run backend and frontend tests"
	@echo "make lint         Check Go, frontend, TypeSpec and generated artifacts"
	@echo "make format       Format Go, frontend and TypeSpec source files"
	@echo "make build        Build backend, frontend, OpenAPI and client SDK"
	@echo "make generate     Generate OpenAPI, the frontend SDK and Go server package"
	@echo "make generate-client  Generate the frontend SDK from TypeSpec"
	@echo "make generate-server  Generate the Go server package from TypeSpec"
	@echo "make lint-server      Check the generated Go server package for drift"
	@echo "make container-build  Build the production application image"
	@echo "make smoke        Smoke-test the containerized production application"
	@echo "make smoke-standalone  Smoke-test the image without an external database"
	@echo "make smoke-remote Smoke-test the deployed application over HTTPS"
	@echo "make check        Run lint, tests, builds and the smoke test"
	@echo "make db-down      Stop PostgreSQL, preserving its data volume"

setup: install tools browser

install:
	cd backend && go mod download
	npm --prefix api ci
	npm --prefix frontend ci

tools: $(GOLANGCI)

browser:
ifeq ($(shell uname -s),Linux)
	npm --prefix frontend exec -- playwright install --with-deps chromium
else
	npm --prefix frontend exec -- playwright install chromium
endif

$(GOLANGCI):
	mkdir -p .bin/$(GOLANGCI_VERSION)
	curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_VERSION)/install.sh -o .bin/$(GOLANGCI_VERSION)/install.sh
	sh .bin/$(GOLANGCI_VERSION)/install.sh -b .bin/$(GOLANGCI_VERSION) $(GOLANGCI_VERSION)

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

migrate:
	cd backend && go run ./cmd/migrate

seed: migrate
	cd backend && go run ./cmd/seed

dev: db-up seed
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && go run ./cmd/api

dev-frontend: lint-client
	npm --prefix frontend run dev

test: test-backend test-frontend

test-backend:
	cd backend && go test -race ./...

test-integration: db-up
	cd backend && go test -race -tags=integration ./internal/storage ./internal/httpapi

test-frontend: lint-client
	npm --prefix frontend test

lint: lint-backend lint-frontend lint-typespec lint-openapi lint-client lint-server lint-render
	node --check scripts/smoke.mjs
	node --check scripts/check-client-sdk.mjs
	node --check scripts/generate-server.mjs

lint-backend: tools
	cd backend && $(GOLANGCI) run ./...
	@files="$$(cd backend && gofmt -l .)"; if [ -n "$$files" ]; then echo "Run make format:"; echo "$$files"; exit 1; fi

lint-frontend: lint-client
	npm --prefix frontend run lint
	npm --prefix frontend run format:check

lint-typespec:
	npm --prefix api run check

lint-openapi: build-typespec
	@git ls-files --error-unmatch api/tsp-output/openapi.yaml >/dev/null 2>&1 && \
		git diff --quiet -- api/tsp-output/openapi.yaml || { \
		echo "Generated OpenAPI is out of date. Run 'make build-typespec' and commit api/tsp-output/openapi.yaml."; \
		exit 1; \
	}

lint-client: build-typespec
	node scripts/check-client-sdk.mjs

lint-server: build-typespec
	node scripts/generate-server.mjs --check

lint-render:
	cd frontend && npx prettier --check ../render.yaml

format:
	cd backend && gofmt -w .
	npm --prefix api run format
	npm --prefix frontend run format
	cd frontend && npx prettier --write ../render.yaml

build: build-backend build-frontend build-typespec

build-backend:
	mkdir -p .bin
	cd backend && go build -o ../.bin/api ./cmd/api

build-frontend: lint-client
	npm --prefix frontend run build

build-typespec:
	npm --prefix api run build

generate: generate-client generate-server

generate-client: build-typespec
	npm --prefix frontend run generate:api

generate-server: build-typespec
	node scripts/generate-server.mjs --write

container-build:
	docker build --target application -t calendar-of-calls:local .

smoke:
	@set -eu; \
	cleanup() { PORT=$(SMOKE_APP_PORT) POSTGRES_PORT=$(SMOKE_POSTGRES_PORT) APP_PORT=$(SMOKE_APP_PORT) docker compose -p $(SMOKE_PROJECT) down --volumes --remove-orphans; }; \
	trap cleanup EXIT INT TERM; \
	cleanup; \
	PORT=$(SMOKE_APP_PORT) POSTGRES_PORT=$(SMOKE_POSTGRES_PORT) APP_PORT=$(SMOKE_APP_PORT) docker compose -p $(SMOKE_PROJECT) up -d --build --wait app; \
	SMOKE_BASE_URL=http://127.0.0.1:$(SMOKE_APP_PORT) node scripts/smoke.mjs; \
	E2E_BASE_URL=http://127.0.0.1:$(SMOKE_APP_PORT) npm --prefix frontend run test:e2e

smoke-standalone:
	@set -eu; \
	cleanup() { docker rm -f $(STANDALONE_SMOKE_CONTAINER) >/dev/null 2>&1 || true; }; \
	trap cleanup EXIT INT TERM; \
	cleanup; \
	docker build -t $(STANDALONE_SMOKE_IMAGE) .; \
	docker run -d --name $(STANDALONE_SMOKE_CONTAINER) -e PORT=8080 -p 127.0.0.1:$(STANDALONE_SMOKE_APP_PORT):8080 $(STANDALONE_SMOKE_IMAGE); \
	SMOKE_BASE_URL=http://127.0.0.1:$(STANDALONE_SMOKE_APP_PORT) node scripts/smoke.mjs || { docker logs $(STANDALONE_SMOKE_CONTAINER); exit 1; }

smoke-remote:
	@test -n "$(REMOTE_SMOKE_BASE_URL)" || { echo "Set REMOTE_SMOKE_BASE_URL to the deployed HTTPS origin"; exit 2; }
	SMOKE_BASE_URL="$(REMOTE_SMOKE_BASE_URL)" SMOKE_REQUIRE_HTTPS=1 node scripts/smoke.mjs

check: lint test test-integration smoke smoke-standalone
