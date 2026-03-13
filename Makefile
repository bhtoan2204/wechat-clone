run:
	@go run cmd/main.go
	@cd gateway && make run
.PHONY: run

up:
	@docker compose up -d
.PHONY: up

down:
	@docker compose down
.PHONY: down

build:
	@docker compose build
.PHONY: build

generate:
	go run scaffold/main.go
.PHONY: generate