include .envrc

.DEFAULT_GOAL := run
.PHONY: run
run: run-docker migrate run-app-local

.PHONY: run-docker
run-docker:
	docker compose up -d

.PHONY: run-app-local
run-app-local:
	go run ./cmd/api -db-dsn=${GREENLIGHT_DB_DSN} -jwt-secret=${JWT_SECRET}

.PHONY: migrate
migrate:
	migrate -path=./migrations -database=${GREENLIGHT_DB_DSN} up

.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy
	go mod verify
	go mod vendor

.PHONY: audit
audit:
	go mod tidy -diff
	go mod verify
	go vet ./...
	staticcheck ./...
	go test -race -vet=off ./...

.PHONY: build/api
build/api:
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api
