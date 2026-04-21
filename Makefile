.DEFAULT_GOAL := help

FULL_COMPOSE := docker compose -f docker-compose.dev-min.yml -f docker-compose.full.yml
APP_COMPOSE := docker compose -f docker-compose.app.yml
PRODUCTION_COMPOSE := docker compose -f docker-compose.dev-min.yml -f docker-compose.full.yml -f docker-compose.app.yml

BASE_ENV_FILE := secret/.env

define LOAD_BASE_ENV
set -a; . ./$(BASE_ENV_FILE); set +a;
endef

## Show available commands
help:
	@printf "Available targets:\n"
	@printf "  make up             Start the local stack with Kafka, Debezium, and Elasticsearch\n"
	@printf "  make up-full        Alias of make up\n"
	@printf "  make down           Stop and remove local containers\n"
	@printf "  make bootstrap      Prepare local Cassandra and MinIO dependencies\n"
	@printf "  make migrate        Apply database migrations without starting the API\n"
	@printf "  make run            Run the API with the full local profile\n"
	@printf "  make run-full       Alias of make run\n"
	@printf "  make run-gateway    Run the optional gateway service\n"
	@printf "  make fmt            Format Go code with goimports\n"
	@printf "  make lint           Run go vet across the repository\n"
	@printf "  make test           Run Go tests\n"
	@printf "  make generate       Regenerate scaffold outputs\n"
	@printf "  make connector      Register Debezium connectors against Kafka Connect\n"
	@printf "  make connector-reset Reset connector offsets in Kafka Connect\n"
.PHONY: help

## Start the full local stack for development
up:
	@bash -c '$(LOAD_BASE_ENV) $(FULL_COMPOSE) up -d'
.PHONY: up

## Alias for the default full local stack
up-full:
	@bash -c '$(LOAD_BASE_ENV) $(PRODUCTION_COMPOSE) up -d'
.PHONY: up-full

up-app:
	@bash -c '$(LOAD_BASE_ENV) $(APP_COMPOSE) up -d'

## Stop and remove local containers
down:
	@bash -c '$(LOAD_BASE_ENV) $(FULL_COMPOSE) down --remove-orphans'
.PHONY: down

down-all:
	@bash -c '$(LOAD_BASE_ENV) $(PRODUCTION_COMPOSE) down --remove-orphans'
.PHONY: down

## Prepare Cassandra keyspace and MinIO bucket for local development
bootstrap:
	@./script/bin.sh bootstrap
.PHONY: bootstrap

## Apply database migrations without starting the HTTP server
migrate:
	@./script/bin.sh migrate
.PHONY: migrate

## Run the API server with the full local development profile
run:
	@./script/bin.sh run
.PHONY: run

## Alias for the default full local development profile
run-full:
	@./script/bin.sh run
.PHONY: run-full

## Run the Consul-aware reverse proxy gateway
run-gateway:
	@cd gateway && make run
.PHONY: run-gateway

## Format Go sources with goimports
fmt:
	@goimports -w .
.PHONY: fmt

## Run lightweight static checks
lint:
	@go vet ./...
.PHONY: lint

## Run repository tests
test:
	@go test ./...
.PHONY: test

## Regenerate scaffolded routes, handlers, and OpenAPI artifacts
generate:
	@go run scaffold/main.go
.PHONY: generate

## Register Debezium connectors after Kafka Connect becomes healthy
connector:
	@CONNECT_URL=$${CONNECT_URL:-http://localhost:8083} CONNECTOR_CONFIG_FILE=./connector/connector_config.json CONNECTOR_NAME=$${CONNECTOR_NAME:-} /bin/sh ./connector/register_connectors.sh
.PHONY: connector

## Reset Debezium connector offsets for replay and rebuild flows
connector-reset:
	@CONNECT_URL=$${CONNECT_URL:-http://localhost:8083} CONNECTOR_NAME=$${CONNECTOR_NAME:-} /bin/sh ./connector/reset_connector_offsets.sh
.PHONY: connector-reset
