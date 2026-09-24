default: @just --list

# Local dev: postgres only (plan §3 — redis/minio/mailpit removed).
deps-up:
    docker compose -f deploy/compose.dev.yaml up -d

deps-down:
    docker compose -f deploy/compose.dev.yaml down

# Run one service locally (needs DATABASE_URL + DOMAIN in env).
dev service="api":
    go run ./cmd/{{service}}

# Codegen: sqlc + OpenAPI (huma) + typed TS client for web/.
gen:
    sqlc generate -f internal/db/sqlc.yaml
    go run ./cmd/api --print-openapi > web/openapi.json
    pnpm --dir web exec openapi-typescript openapi.json -o lib/api.d.ts

# DB migrations (goose, postgres only).
migrate:
    goose -dir internal/db/migrations postgres "$DATABASE_URL" up

migrate-down:
    goose -dir internal/db/migrations postgres "$DATABASE_URL" down

seed:
    go run ./scripts/seed

test:
    go test -race ./... -count=1

test-unit:
    go test -race $(go list ./... | grep -v /internal/db) -count=1

lint:
    golangci-lint run ./...
    go vet ./...
    gitleaks detect --no-git --redact || true

fmt:
    gofumpt -w .
    gofmt -w .

sec:
    gitleaks detect --no-git --redact
    govulncheck ./...
    trivy fs --severity HIGH,CRITICAL .

# End-to-end against the real Incus on the host (plan §6).
e2e:
    ./scripts/e2e.sh

# CI parity: generated OpenAPI/TS client must be up to date.
ci-check:
    sqlc diff -f internal/db/sqlc.yaml
    go run ./cmd/api --print-openapi > /tmp/openapi.json
    diff -q /tmp/openapi.json web/openapi.json
    pnpm --dir web exec openapi-typescript /tmp/openapi.json -o /tmp/api.d.ts
    diff -q /tmp/api.d.ts web/lib/api.d.ts
    go test -race ./... -count=1
