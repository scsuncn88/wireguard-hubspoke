# WireGuard SD-WAN Makefile

# Build configuration
GO_VERSION = 1.21
BINARY_NAME = wireguard-sdwan
VERSION ?= $(shell git describe --tags --always --dirty)
BUILD_TIME = $(shell date -u +%Y%m%d%H%M%S)
COMMIT_HASH = $(shell git rev-parse --short HEAD)

# Go build flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commitHash=$(COMMIT_HASH)"

# Directories
BUILD_DIR = build
DIST_DIR = dist
DOCKER_DIR = infra/docker

# Default target
.PHONY: all
all: clean build test

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	go clean -cache

# Build all components
.PHONY: build
build: build-controller build-agent build-cli build-ui

# Build controller
.PHONY: build-controller
build-controller:
	@echo "Building controller..."
	@mkdir -p $(BUILD_DIR)
	cd controller && go build $(LDFLAGS) -o ../$(BUILD_DIR)/controller main.go

# Build agent
.PHONY: build-agent
build-agent:
	@echo "Building agent..."
	@mkdir -p $(BUILD_DIR)
	cd agent && go build $(LDFLAGS) -o ../$(BUILD_DIR)/agent main.go

# Build CLI
.PHONY: build-cli
build-cli:
	@echo "Building CLI..."
	@mkdir -p $(BUILD_DIR)
	cd cli && go build $(LDFLAGS) -o ../$(BUILD_DIR)/wg-sdwan-cli main.go

# Build UI
.PHONY: build-ui
build-ui:
	@echo "Building UI..."
	cd ui && npm ci && npm run build

# Development setup
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	@if [ ! -f .env ]; then cp .env.example .env; fi
	@if [ ! -d venv_linux ]; then python3 -m venv venv_linux; fi
	@source venv_linux/bin/activate && pip install -r requirements.txt || true
	cd ui && npm install

# Start development environment
.PHONY: dev-start
dev-start:
	@echo "Starting development environment..."
	docker-compose -f $(DOCKER_DIR)/docker-compose.dev.yml up -d

# Stop development environment
.PHONY: dev-stop
dev-stop:
	@echo "Stopping development environment..."
	docker-compose -f $(DOCKER_DIR)/docker-compose.dev.yml down

# Database migrations
.PHONY: db-migrate
db-migrate:
	@echo "Running database migrations..."
	cd controller && go run migrations/migrate.go up

# Database rollback
.PHONY: db-rollback
db-rollback:
	@echo "Rolling back database..."
	cd controller && go run migrations/migrate.go down

# Testing
.PHONY: test
test: test-unit test-integration

# Unit tests
.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out ./...

# Integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	go test -v -tags=integration ./tests/integration/...

# End-to-end tests
.PHONY: test-e2e
test-e2e:
	@echo "Running end-to-end tests..."
	cd tests/e2e && npm test

# Code quality
.PHONY: lint
lint: lint-go lint-ui

# Go linting
.PHONY: lint-go
lint-go:
	@echo "Running Go linting..."
	golangci-lint run ./...

# UI linting
.PHONY: lint-ui
lint-ui:
	@echo "Running UI linting..."
	cd ui && npm run lint

# Code formatting
.PHONY: format
format: format-go format-ui

# Go formatting
.PHONY: format-go
format-go:
	@echo "Formatting Go code..."
	gofmt -w .
	goimports -w .

# UI formatting
.PHONY: format-ui
format-ui:
	@echo "Formatting UI code..."
	cd ui && npm run format

# Security scanning
.PHONY: security-scan
security-scan:
	@echo "Running security scan..."
	gosec ./...
	cd ui && npm audit

# Generate code
.PHONY: generate
generate:
	@echo "Generating code..."
	go generate ./...

# Build Docker images
.PHONY: docker-build
docker-build:
	@echo "Building Docker images..."
	docker build -t $(BINARY_NAME)-controller:$(VERSION) -f $(DOCKER_DIR)/Dockerfile.controller .
	docker build -t $(BINARY_NAME)-agent:$(VERSION) -f $(DOCKER_DIR)/Dockerfile.agent .
	docker build -t $(BINARY_NAME)-ui:$(VERSION) -f $(DOCKER_DIR)/Dockerfile.ui .

# Push Docker images
.PHONY: docker-push
docker-push:
	@echo "Pushing Docker images..."
	docker push $(BINARY_NAME)-controller:$(VERSION)
	docker push $(BINARY_NAME)-agent:$(VERSION)
	docker push $(BINARY_NAME)-ui:$(VERSION)

# Deploy to development
.PHONY: deploy-dev
deploy-dev:
	@echo "Deploying to development..."
	docker-compose -f $(DOCKER_DIR)/docker-compose.dev.yml up -d --build

# Deploy to production
.PHONY: deploy-prod
deploy-prod:
	@echo "Deploying to production..."
	docker-compose -f $(DOCKER_DIR)/docker-compose.prod.yml up -d --build

# Kubernetes deployment
.PHONY: k8s-deploy
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	helm upgrade --install wireguard-sdwan ./infra/helm/ \
		--set image.tag=$(VERSION) \
		--namespace wireguard-sdwan \
		--create-namespace

# Create release
.PHONY: release
release: clean build test
	@echo "Creating release $(VERSION)..."
	@mkdir -p $(DIST_DIR)
	@tar -czf $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BUILD_DIR) .
	@echo "Release created: $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz"

# Install tools
.PHONY: tools
tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/swaggo/swag/cmd/swag@latest

# Generate API documentation
.PHONY: docs
docs:
	@echo "Generating API documentation..."
	cd controller && swag init
	@echo "API documentation generated in controller/docs/"

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all           - Clean, build, and test everything"
	@echo "  build         - Build all components"
	@echo "  build-*       - Build specific component (controller, agent, cli, ui)"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run all tests"
	@echo "  test-*        - Run specific test suite (unit, integration, e2e)"
	@echo "  lint          - Run all linters"
	@echo "  format        - Format all code"
	@echo "  security-scan - Run security scans"
	@echo "  dev-setup     - Set up development environment"
	@echo "  dev-start     - Start development environment"
	@echo "  dev-stop      - Stop development environment"
	@echo "  db-migrate    - Run database migrations"
	@echo "  docker-build  - Build Docker images"
	@echo "  docker-push   - Push Docker images"
	@echo "  deploy-*      - Deploy to environment (dev, prod)"
	@echo "  k8s-deploy    - Deploy to Kubernetes"
	@echo "  release       - Create release package"
	@echo "  tools         - Install development tools"
	@echo "  docs          - Generate API documentation"
	@echo "  help          - Show this help"