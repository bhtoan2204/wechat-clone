run:
	@echo "Formatting code..."
	@goimports -w .
	@script/bin.sh run
.PHONY: run

run-gateway:
	@cd gateway && make run
.PHONY: run-gateway

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
	go run scaffold/main.go && go generate ./...
.PHONY: generate

connector:
	@CONNECT_URL=$${CONNECT_URL:-http://localhost:8083} CONNECTOR_CONFIG_FILE=./connector/connector_config.json CONNECTOR_NAME=$${CONNECTOR_NAME:-} /bin/sh ./connector/register_connectors.sh
.PHONY: connector

connector-reset:
	@CONNECT_URL=$${CONNECT_URL:-http://localhost:8083} CONNECTOR_NAME=$${CONNECTOR_NAME:-} /bin/sh ./connector/reset_connector_offsets.sh
.PHONY: connector-reset
