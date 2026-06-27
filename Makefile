.PHONY: help migrate-up migrate-down migrate-create sqlc test integration-test

help:
	@echo "Targets:"
	@echo "  migrate-up        Apply pending migrations"
	@echo "  migrate-down      Roll back last migration"
	@echo "  migrate-create    Create a new migration file (NAME=...)"
	@echo "  sqlc              Regenerate internal/db from db/queries"
	@echo "  test              Run unit tests"
	@echo "  integration-test  Run integration tests (requires test DB)"

migrate-up:
	@echo "TODO: wire goose or golang-migrate"

migrate-down:
	@echo "TODO: wire goose or golang-migrate"

migrate-create:
	@echo "TODO: wire goose or golang-migrate (NAME=$(NAME))"

sqlc:
	sqlc generate

test:
	go test ./...

integration-test:
	go test ./test/integration/...
