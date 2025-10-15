// ============================================================================
// FILE: Makefile
// ============================================================================
.PHONY: help setup build run test test-integration test-coverage clean redis-up redis-down docker-build docker-up docker-down lint

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install dependencies and setup development environment
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Setup complete!"

build: ## Build the application
	@echo "Building application..."
	go build -o bin/server cmd/server/main.go
	@echo "Build complete! Binary: bin/server"

run: ## Run the application
	@echo "Starting server..."
	go run cmd/server/main.go

test: ## Run unit tests
	@echo "Running tests..."
	go test -v -race -timeout 30s ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -race -tags=integration ./...

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	@echo "Clean complete!"

redis-up: ## Start Redis in Docker
	@echo "Starting Redis..."
	docker run -d --name peoplearoundme-redis -p 6379:6379 redis:7-alpine
	@echo "Redis started on port 6379"

redis-down: ## Stop Redis Docker container
	@echo "Stopping Redis..."
	docker stop peoplearoundme-redis || true
	docker rm peoplearoundme-redis || true
	@echo "Redis stopped"

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t peoplearoundme:latest .
	@echo "Docker image built!"

docker-up: ## Start application with Docker Compose
	@echo "Starting application..."
	docker-compose up -d
	@echo "Application started!"

docker-down: ## Stop Docker Compose
	@echo "Stopping application..."
	docker-compose down
	@echo "Application stopped"

lint: ## Run linter
	@echo "Running linter..."
	golangci-lint run ./...

.DEFAULT_GOAL := help