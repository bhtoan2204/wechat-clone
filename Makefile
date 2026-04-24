.DEFAULT_GOAL := help

INFRA_COMPOSE := docker compose -f docker-compose.infra.yml
APP_COMPOSE := docker compose -f docker-compose.app.yml
FULL_COMPOSE := docker compose -f docker-compose.infra.yml -f docker-compose.app.yml

BASE_ENV_FILE := secret/.env

define LOAD_BASE_ENV
set -a; . ./$(BASE_ENV_FILE); set +a;
endef

## Show available commands
help:
	@printf "Available targets:\n"
	@printf "  make up             Start the local infra stack\n"
	@printf "  make up-infra       Alias of make up\n"
	@printf "  make up-ui          Start optional infra UIs and admin tools\n"
	@printf "  make up-app         Start app services only\n"
	@printf "  make up-full        Start infra + app stack\n"
	@printf "  make down           Stop infra containers\n"
	@printf "  make down-infra     Alias of make down\n"
	@printf "  make down-app       Stop app containers\n"
	@printf "  make down-all       Stop infra + app containers\n"
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

## Start the local infra stack
up:
	@bash -c '$(LOAD_BASE_ENV) $(INFRA_COMPOSE) up -d --build'
.PHONY: up

## Alias for infra stack
up-infra:
	@bash -c '$(LOAD_BASE_ENV) $(INFRA_COMPOSE) up -d --build'
.PHONY: up-infra

## Start optional infra UIs and admin tools
up-ui:
	@bash -c '$(LOAD_BASE_ENV) COMPOSE_PROFILES=ops-ui $(INFRA_COMPOSE) up -d --build'
.PHONY: up-ui

## Start app stack only
up-app:
	@bash -c '$(LOAD_BASE_ENV) $(APP_COMPOSE) up -d --build'
.PHONY: up-app

## Start infra + app stack
up-full:
	@bash -c '$(LOAD_BASE_ENV) $(FULL_COMPOSE) up -d --build'
.PHONY: up-full

## Stop infra stack
down:
	@bash -c '$(LOAD_BASE_ENV) $(INFRA_COMPOSE) down'
.PHONY: down

## Stop infra stack
down-infra:
	@bash -c '$(LOAD_BASE_ENV) $(INFRA_COMPOSE) down'
.PHONY: down-infra

## Stop app stack
down-app:
	@bash -c '$(LOAD_BASE_ENV) $(APP_COMPOSE) down'
.PHONY: down-app

## Stop infra + app stack
down-all:
	@bash -c '$(LOAD_BASE_ENV) $(FULL_COMPOSE) down --remove-orphans'
.PHONY: down-all

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
