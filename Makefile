.PHONY: help build test lint staticcheck check install clean docs

BINARY_NAME=go-arch-lint
INSTALL_PATH=$(shell go env GOPATH)/bin/$(BINARY_NAME)

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_-]+:.*?##/ { printf "  %-15s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: ## Build the binary
	go build -o $(BINARY_NAME) ./cmd/go-arch-lint

init: build ## Initialize a new project with default config and docs
	./$(BINARY_NAME) init

test: ## Run all tests
	go test ./...

lint: build ## Run go-arch-lint on itself (must show zero violations)
	./$(BINARY_NAME) .

staticcheck: ## Run staticcheck on all packages
	@which staticcheck > /dev/null || (echo "Installing staticcheck..." && go install honnef.co/go/tools/cmd/staticcheck@latest)
	staticcheck ./...

check: test lint staticcheck ## Run all tests and linters

install: ## Install binary to GOPATH/bin
	go install ./cmd/go-arch-lint
	@echo "Installed to: $(INSTALL_PATH)"

clean: ## Remove built binary
	rm -f $(BINARY_NAME)

docs: build ## Generate documentation files
	./$(BINARY_NAME) -detailed -format=markdown . > docs/arch-generated.md 2>&1
	./$(BINARY_NAME) -format=api . > docs/public-api-generated.md 2>&1
	@echo "Documentation updated"
