BINARY_NAME = myapp
OUTPUT_DIR = bin

GOBUILD=go build -o ${OUTPUT_DIR}/${BINARY_NAME}

# Цель по умолчанию
.DEAFULT_GOAL := help

help: 
	@echo "Use: make <target>"
	@echo "Targets:"
	@echo " lint - check cyclomatic complexities"
	@echo " deps - update dependencies"
	@echo " build - собрать проект"
	@echo " run   - запустить проект"
	@echo " clean - удалить проект"
	
lint: 
	@echo "Linting..."
	@echo "Check cyclomatic complexity (gocyclo)..."
	@gocyclo -over 10 .
	@echo "Check cognitive complexity (gocognit)..."
	@gocognit -over 10 .
	
deps: 
	@echo "Updating packages..."
	@go mod tidy
	@go mod download
	
build: deps lint
	@echo "Building..."
	@mkdir -p $(OUTPUT_DIR)
	@${GOBUILD}
	
run: build 
	@echo "Running..."
	@./$(OUTPUT_DIR)/$(BINARY_NAME)
	
clean: 
	@echo "Cleaning..."
	rm -rf $(OUTPUT_DIR)
	@echo "Файл удален"