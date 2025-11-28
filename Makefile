BINARY_NAME = myapp
OUTPUT_DIR = bin

GOBUILD=go build -o ${OUTPUT_DIR}/${BINARY_NAME}

.DEAFULT_GOAL := help

## Load environment file if present (defines DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, DB_SSLMODE)
ifneq (,$(wildcard .env))
	include .env
endif

# Provide defaults when variables are not set in environment/.env
DB_HOST ?= localhost
DB_PORT ?= 5432
DB_USER ?= test
DB_PASSWORD ?= test
DB_NAME ?= test
DB_SSLMODE ?= disable

POSTGRES_SETUP_TEST ?= user=$(DB_USER) password=$(DB_PASSWORD) dbname=$(DB_NAME) host=$(DB_HOST) port=$(DB_PORT) sslmode=$(DB_SSLMODE)

MIGRATION_FOLDER=$(CURDIR)/db/migrations

.PHONY: migration-create
migration-create:
	goose -dir "$(MIGRATION_FOLDER)" create "$(name)" sql

.PHONY: test-migration-up
test-migration-up:
	goose -dir "$(MIGRATION_FOLDER)" postgres "$(POSTGRES_SETUP_TEST)" up

.PHONY: test-migration-down
test-migration-down:
	goose -dir "$(MIGRATION_FOLDER)" postgres "$(POSTGRES_SETUP_TEST)" down

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