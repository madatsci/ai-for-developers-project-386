.DEFAULT_GOAL := help

-include .env

export DATABASE_URL ?= postgres://calendar:calendar@127.0.0.1:5433/calendar?sslmode=disable
export HTTP_ADDR ?= 127.0.0.1:8080
export API_PROXY_TARGET ?= http://127.0.0.1:8080
export FRONTEND_PORT ?= 5175

GOLANGCI_VERSION := v2.15.0
GOLANGCI := $(CURDIR)/.bin/$(GOLANGCI_VERSION)/golangci-lint

.PHONY: help setup install tools db-up db-down dev dev-backend dev-frontend test test-backend test-frontend lint lint-backend lint-frontend format build build-backend build-frontend smoke check

help:
	@echo "make setup        Install dependencies and the pinned Go linter"
	@echo "make dev          Start PostgreSQL, backend and frontend"
	@echo "make test         Run backend and frontend tests"
	@echo "make lint         Check Go, ESLint and Prettier"
	@echo "make format       Format Go and frontend source files"
	@echo "make build        Build backend and frontend"
	@echo "make smoke        Start PostgreSQL and smoke-test production builds"
	@echo "make check        Run lint, tests, builds and the smoke test"
	@echo "make db-down      Stop PostgreSQL, preserving its data volume"

setup: install tools

install:
	cd backend && go mod download
	npm --prefix frontend ci

tools: $(GOLANGCI)

$(GOLANGCI):
	mkdir -p .bin/$(GOLANGCI_VERSION)
	curl -fsSL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_VERSION)/install.sh -o .bin/$(GOLANGCI_VERSION)/install.sh
	sh .bin/$(GOLANGCI_VERSION)/install.sh -b .bin/$(GOLANGCI_VERSION) $(GOLANGCI_VERSION)

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

dev: db-up
	$(MAKE) -j2 dev-backend dev-frontend

dev-backend:
	cd backend && go run ./cmd/api

dev-frontend:
	npm --prefix frontend run dev

test: test-backend test-frontend

test-backend:
	cd backend && go test -race ./...

test-frontend:
	npm --prefix frontend test

lint: lint-backend lint-frontend
	node --check scripts/smoke.mjs

lint-backend: tools
	cd backend && $(GOLANGCI) run ./...
	@files="$$(cd backend && gofmt -l .)"; if [ -n "$$files" ]; then echo "Run make format:"; echo "$$files"; exit 1; fi

lint-frontend:
	npm --prefix frontend run lint
	npm --prefix frontend run format:check

format:
	cd backend && gofmt -w .
	npm --prefix frontend run format

build: build-backend build-frontend

build-backend:
	mkdir -p .bin
	cd backend && go build -o ../.bin/api ./cmd/api

build-frontend:
	npm --prefix frontend run build

smoke: db-up build
	node scripts/smoke.mjs

check: lint test smoke
