.PHONY: help dev worker sms-test migrate-up migrate-down migrate-create migrate-status sqlc test integration-test

-include .env
export

DB_URL ?= postgres://amirabaris@localhost:5432/weeto?sslmode=disable

help:
	@echo "Targets:"
	@echo "  dev               Run API with hot reload (air)"
	@echo "  worker            Run notification outbox worker"
	@echo "  sms-test          Send sandbox SMS (PHONE=0912...)"
	@echo "  migrate-up        Apply pending migrations"
	@echo "  migrate-down      Roll back last migration"
	@echo "  migrate-create    Create a new migration file (NAME=...)"
	@echo "  migrate-status    Show migration status"
	@echo "  sqlc              Regenerate internal/db from db/queries"
	@echo "  test              Run unit tests"
	@echo "  integration-test  Run integration tests (requires test DB)"

dev:
	go run github.com/air-verse/air@latest

worker:
	go run ./cmd/worker

sms-test:
	@test -n "$(PHONE)" || (echo "usage: make sms-test PHONE=0912XXXXXXX" && exit 1)
	go run ./cmd/sms-test -phone "$(PHONE)"

migrate-up:
	goose -dir db/migrations postgres "$(DB_URL)" up

migrate-down:
	goose -dir db/migrations postgres "$(DB_URL)" down

migrate-create:
	@test -n "$(NAME)" || (echo "usage: make migrate-create NAME=add_users" && exit 1)
	goose -dir db/migrations create $(NAME) sql

migrate-status:
	goose -dir db/migrations postgres "$(DB_URL)" status

sqlc:
	sqlc generate

test:
	go test ./...

integration-test:
	go test ./test/integration/...
