BINARY_NAME = myapp
OUTPUT_DIR = bin

GOBUILD=go build -o ${OUTPUT_DIR}/${BINARY_NAME}
GOPATH ?= $(shell go env GOPATH)
GOMODCACHE ?= $(shell go env GOMODCACHE)
GRPC_GATEWAY_V2_DIR ?= $(GOMODCACHE)/github.com/grpc-ecosystem/grpc-gateway/v2@v2.29.0

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

.PHONY: generate-http-api
generate-http-api:
	go generate -run=oapi-codegen ./...

.PHONY: generate-grpc-api
generate-grpc-api:
	mkdir -p pkg/api
	protoc \
		-I . \
		-I pkg \
		-I "$(GRPC_GATEWAY_V2_DIR)" \
		--go_out=pkg/api \
		--go-grpc_out=pkg/api \
		cmd/api/proto/common.proto \
		cmd/api/proto/orders.proto

.PHONY: generate-grpc-gateway-api
generate-grpc-gateway-api:
	mkdir -p pkg/api cmd/api/openapi/orders/generated
	protoc \
		-I . \
		-I pkg \
		-I "$(GRPC_GATEWAY_V2_DIR)" \
		--go_out=pkg/api \
		--go-grpc_out=pkg/api \
		--grpc-gateway_out=pkg/api \
		--openapiv2_out=cmd/api/openapi/orders/generated \
		--openapiv2_opt=output_format=yaml,json_names_for_fields=true,simple_operation_ids=true,preserve_rpc_order=true,allow_merge=true,merge_file_name=orders,openapi_naming_strategy=simple,disable_default_errors=true,disable_default_responses=true \
		cmd/api/proto/common.proto \
		cmd/api/proto/orders.proto

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
