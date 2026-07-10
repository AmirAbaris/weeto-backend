.PHONY: help dev worker migrate-up migrate-down migrate-create migrate-status sqlc test integration-test docker-up docker-down docker-logs docker-prod-up docker-migrate

-include .env
export

DB_URL ?= postgres://amirabaris@localhost:5432/weeto?sslmode=disable

help:
	@echo "Targets:"
	@echo "  dev               Run API with hot reload (air)"
	@echo "  worker            Run notification outbox worker"
	@echo "  migrate-up        Apply pending migrations"
	@echo "  migrate-down      Roll back last migration"
	@echo "  migrate-create    Create a new migration file (NAME=...)"
	@echo "  migrate-status    Show migration status"
	@echo "  sqlc              Regenerate internal/db from db/queries"
	@echo "  test              Run unit tests"
	@echo "  integration-test  Run integration tests (requires test DB)"
	@echo "  docker-up         Build and start full stack (postgres + migrate + api)"
	@echo "  docker-down       Stop docker compose stack"
	@echo "  docker-logs       Follow API container logs"
	@echo "  docker-prod-up    Build and start production stack"
	@echo "  docker-migrate    Run migrations in docker"

dev:
	go run github.com/air-verse/air@latest

worker:
	go run ./cmd/worker

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

docker-up:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d --build

docker-down:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml down

docker-logs:
	docker compose -f docker-compose.yml -f docker-compose.dev.yml logs -f api

docker-prod-up:
	docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --build

docker-migrate:
	docker compose -f docker-compose.yml run --rm migrate
