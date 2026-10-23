# Repository Guidelines

## Project Structure & Module Organization

- `backend/cmd/api/` starts the HTTP server and database pool.
- `backend/internal/httpapi/` contains API handlers and colocated Go tests.
- `frontend/src/` contains Vue components, views, routing, assets, and frontend tests.
- `scripts/smoke.mjs` validates the built application end to end.
- `compose.yaml` provides the local PostgreSQL service; `.github/workflows/` defines CI and releases.

Keep domain logic close to its backend package or frontend feature. Do not commit `.bin/`, `frontend/dist/`, or coverage output.

## Build, Test, and Development Commands

Run commands from the repository root:

- `make setup` installs Go/npm dependencies and the pinned golangci-lint binary.
- `make dev` starts PostgreSQL, the API on `127.0.0.1:8080`, and Vite on `127.0.0.1:5175`.
- `make test` runs Go tests with the race detector and Vitest once.
- `make lint` runs golangci-lint, gofmt checks, ESLint, Prettier, and a smoke-script syntax check.
- `make format` applies gofmt and Prettier.
- `make build` produces `.bin/api` and `frontend/dist/`.
- `make check` runs the complete CI-equivalent suite, including the PostgreSQL-backed smoke test.

## Coding Style & Naming Conventions

Write idiomatic Go and always run `gofmt`. Use lowercase package names, exported `PascalCase` identifiers, and concise receiver names. Frontend code uses TypeScript and Vue Single-File Components. Prettier enforces no semicolons, single quotes, and a 100-column width; use `PascalCase.vue` for components and descriptive camelCase names for functions and variables.

## Testing Guidelines

Add Go tests as `*_test.go`, preferably table-driven for multiple cases. Place frontend tests under `frontend/src/__tests__/` and name them `*.spec.ts`; use Vue Test Utils with Vitest. Browser E2E tests live under `frontend/e2e/` and use Playwright against the production build. Tests must be deterministic and should not require Docker unless they are explicit integration or smoke tests. Run `make test` during development and `make check` before opening a pull request.

## Commit & Pull Request Guidelines

Use Conventional Commits, as established by `feat: add application skeleton with CI and release automation`. Typical prefixes are `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, and `chore:`. Keep each commit focused.

Pull requests should explain the change and motivation, link the relevant issue, and list verification performed. Include screenshots for UI changes and call out configuration, database, or API-contract changes. Ensure CI passes before review; do not edit `.github/workflows/hexlet-check.yml`.

## Configuration & Security

Copy `.env.example` only when overriding local defaults. Never commit `.env`, credentials, tokens, or production database URLs. The bundled `calendar/calendar` credentials are for local development only.

## Agent skills

### Issue tracker

Issues are tracked in GitHub Issues. See `docs/agents/issue-tracker.md`.

### Triage labels

The default five-label triage vocabulary is used. See `docs/agents/triage-labels.md`.

### Domain docs

This repository uses the single-context domain documentation layout. See `docs/agents/domain.md`.
