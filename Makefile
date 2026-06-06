BINARY_NAME = myapp
OUTPUT_DIR = bin

GOBUILD=go build -o ${OUTPUT_DIR}/${BINARY_NAME}

.DEFAULT_GOAL := help

## Load environment file if present (defines DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE)
ifneq (,$(wildcard .env.local))
	include .env.local
endif

# Provide defaults when variables are not set in environment/.env
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= test
DB_PASSWORD ?= test
DB_NAME ?= test
DB_SSLMODE ?= disable

GOOSE_DBSTRING ?= user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) host=$(DB_HOST) port=$(DB_PORT) sslmode=$(DB_SSLMODE)

MIGRATIONS_DIR=$(CURDIR)/internal/infra/migrations

.PHONY: migration-create
migration-create:
	goose -dir "$(MIGRATIONS_DIR)" create "$(name)" sql

.PHONY: migration-up
migration-up:
	goose -dir "$(MIGRATIONS_DIR)" postgres "$(GOOSE_DBSTRING)" up

.PHONY: migration-down
migration-down:
	goose -dir "$(MIGRATIONS_DIR)" postgres "$(GOOSE_DBSTRING)" down
	
.PHONY: generate-orders-api
generate-orders-api:
	mkdir -p pkg/api
	protoc --go_out=pkg/api --go-grpc_out=pkg/api cmd/api/orders.proto

.PHONY: generate-mockgen
generate-mockgen:
	go generate -run=mockgen ./...

.PHONY: test
test:
	$(info running tests...)
	go test ./...

.PHONY: seed
seed:
	$(info seeding database...)
	go run ./cmd/seed/main.go

.PHONY: help
help: 
	@echo "Use: make <target>"
	@echo "Targets:"
	@echo " lint - check cyclomatic complexities"
	@echo " deps - update dependencies"
	@echo " build - собрать проект"
	@echo " run   - запустить проект"
	@echo " clean - удалить проект"
	
.PHONY: lint
lint: 
	@echo "Linting..."
	@echo "Check cyclomatic complexity (gocyclo)..."
	@gocyclo -over 10 .
	@echo "Check cognitive complexity (gocognit)..."
	@gocognit -over 10 .

.PHONY: deps	
deps: 
	@echo "Updating packages..."
	@go mod tidy
	@go mod download

.PHONY: build		
build: deps lint
	@echo "Building..."
	@mkdir -p $(OUTPUT_DIR)
	@${GOBUILD}

.PHONY: run		
run: build 
	@echo "Running..."
	@./$(OUTPUT_DIR)/$(BINARY_NAME)

.PHONY: clean		
clean: 
	@echo "Cleaning..."
	rm -rf $(OUTPUT_DIR)
	@echo "Файл удален"
