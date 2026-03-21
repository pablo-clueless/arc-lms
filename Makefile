.PHONY: build run dev test clean docker-build docker-up docker-down migrate-up migrate-down migrate-create lint

# Load .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GORUN=$(GOCMD) run
GOTEST=$(GOCMD) test
GOMOD=$(GOCMD) mod
BINARY_NAME=arc-lms
MAIN_PATH=./cmd/api

# Database URL (override with environment variable)
DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/lms?sslmode=disable
MIGRATIONS_PATH = ./migrations

# Build the application
build:
	$(GOBUILD) -o bin/$(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: build
	./bin/$(BINARY_NAME)

# Run with hot reload using Air
dev:
	air

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Clean build files
clean:
	rm -rf bin/
	rm -rf tmp/
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Install golang-migrate CLI
migrate-install:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run all up migrations
migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" -verbose up

# Rollback last migration
migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" -verbose down 1

# Rollback all migrations
migrate-down-all:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" -verbose down

# Show migration version
migrate-version:
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

# Force migration version (use with caution)
migrate-force:
	@read -p "Enter version to force: " version; \
	migrate -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $$version

# Create new migration
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir $(MIGRATIONS_PATH) -seq $$name

# Build Docker image
docker-build:
	docker build -f build/docker/Dockerfile -t arc-lms .

# Start Docker containers
docker-up:
	docker-compose up -d

# Stop Docker containers
docker-down:
	docker-compose down

# View Docker logs
docker-logs:
	docker-compose logs -f

# Start only the database
db-up:
	docker-compose up -d db

# Stop the database
db-down:
	docker-compose down db

# Run linter
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GOCMD) fmt ./...

# Generate swagger docs (if using swag)
swagger:
	swag init -g cmd/api/main.go -o api/swagger

# Setup: install dependencies and run migrations
setup: deps migrate-install db-up
	@echo "Waiting for database to be ready..."
	@sleep 3
	$(MAKE) migrate-up

# Help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  run             - Build and run the application"
	@echo "  dev             - Run with hot reload (requires Air)"
	@echo "  test            - Run tests"
	@echo "  test-coverage   - Run tests with coverage report"
	@echo "  clean           - Remove build artifacts"
	@echo "  deps            - Download and tidy dependencies"
	@echo ""
	@echo "Migrations:"
	@echo "  migrate-install - Install golang-migrate CLI"
	@echo "  migrate-up      - Run all pending migrations"
	@echo "  migrate-down    - Rollback last migration"
	@echo "  migrate-down-all- Rollback all migrations"
	@echo "  migrate-version - Show current migration version"
	@echo "  migrate-create  - Create new migration files"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-up       - Start Docker containers"
	@echo "  docker-down     - Stop Docker containers"
	@echo "  docker-logs     - View Docker logs"
	@echo "  db-up           - Start only database container"
	@echo ""
	@echo "  setup           - Full setup: deps + migrations"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  help            - Show this help"
