# InfraSense Platform Makefile

VERSION     ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME  ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS     := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
REGISTRY    ?= ghcr.io/infrasense

COLLECTORS := ipmi-collector redfish-collector snmp-collector proxmox-collector

.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Print available targets
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: build
build: build-backend build-collectors ## Build all Go binaries

.PHONY: build-backend
build-backend: ## Build the backend API server
	@echo "Building backend..."
	cd backend && go build $(LDFLAGS) -o bin/infrasense-api ./cmd/server/main.go

.PHONY: build-collectors
build-collectors: ## Build all collectors and notification service
	@echo "Building collectors..."
	@for c in $(COLLECTORS); do \
		echo "  Building $$c..."; \
		cd collectors/$$c && go build $(LDFLAGS) -o bin/$$c ./cmd/main.go && cd ../..; \
	done
	@echo "  Building notification-service..."
	cd notification-service && go build $(LDFLAGS) -o bin/notification-service ./cmd/main.go

.PHONY: build-frontend
build-frontend: ## Build the React frontend
	@echo "Building frontend..."
	cd frontend && npm run build

##@ Test

.PHONY: test
test: test-backend test-collectors ## Run all Go tests

.PHONY: test-backend
test-backend: ## Run backend tests
	@echo "Testing backend..."
	cd backend && go test ./...

.PHONY: test-collectors
test-collectors: ## Run collector tests
	@echo "Testing collectors..."
	@for c in $(COLLECTORS); do \
		echo "  Testing $$c..."; \
		cd collectors/$$c && go test ./... && cd ../..; \
	done
	cd notification-service && go test ./...

##@ Lint

.PHONY: lint
lint: ## Run golangci-lint on all Go modules
	@echo "Linting backend..."
	cd backend && golangci-lint run ./...
	@echo "Linting collectors..."
	@for c in $(COLLECTORS); do \
		echo "  Linting $$c..."; \
		cd collectors/$$c && golangci-lint run ./... && cd ../..; \
	done
	@echo "Linting notification-service..."
	cd notification-service && golangci-lint run ./...

##@ Docker

.PHONY: docker-build
docker-build: ## Build all Docker images
	@echo "Building Docker images..."
	docker build -t $(REGISTRY)/infrasense-api:$(VERSION) \
		--build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) \
		backend/
	@for c in $(COLLECTORS); do \
		echo "  Building $$c image..."; \
		docker build -t $(REGISTRY)/$$c:$(VERSION) \
			--build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) \
			collectors/$$c/; \
	done
	docker build -t $(REGISTRY)/infrasense-notification:$(VERSION) \
		--build-arg VERSION=$(VERSION) --build-arg BUILD_TIME=$(BUILD_TIME) \
		notification-service/

.PHONY: docker-push
docker-push: ## Push Docker images (requires REGISTRY env var)
	@if [ -z "$(REGISTRY)" ]; then echo "Error: REGISTRY is not set"; exit 1; fi
	@echo "Pushing Docker images to $(REGISTRY)..."
	docker push $(REGISTRY)/infrasense-api:$(VERSION)
	@for c in $(COLLECTORS); do \
		docker push $(REGISTRY)/$$c:$(VERSION); \
	done
	docker push $(REGISTRY)/infrasense-notification:$(VERSION)

##@ Packaging

.PHONY: package-deb
package-deb: ## Build the .deb package
	@echo "Building .deb package..."
	bash packaging/scripts/build_deb.sh

.PHONY: package-rpm
package-rpm: ## Build the .rpm package
	@echo "Building .rpm package..."
	bash packaging/scripts/build_rpm.sh

.PHONY: package-all
package-all: package-deb package-rpm ## Build both .deb and .rpm packages

##@ Clean

.PHONY: clean
clean: ## Remove build artifacts
	@echo "Cleaning build artifacts..."
	rm -f backend/bin/*
	@for c in $(COLLECTORS); do \
		rm -f collectors/$$c/bin/*; \
	done
	rm -f notification-service/bin/*
	rm -rf frontend/dist frontend/build
	rm -f dist/*.deb dist/*.rpm
	@echo "Done."
