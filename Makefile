GREENLIGHT_DB_DSN := postgres://greenlight:pa55word@localhost/greenlight?sslmode=disable

run: run-docker migrate run-app-local

run-docker:
	docker compose up -d
run-app-local:
	go run ./cmd/api
migrate:
	migrate -path=./migrations -database=${GREENLIGHT_DB_DSN} up

.DEFAULT_GOAL := run
