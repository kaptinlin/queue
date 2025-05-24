# Set up GOBIN so that our binaries are installed to ./bin instead of $GOPATH/bin.
PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
export GOBIN = $(PROJECT_ROOT)/bin

GOLANGCI_LINT_VERSION := $(shell $(GOBIN)/golangci-lint version --format short 2>/dev/null)
REQUIRED_GOLANGCI_LINT_VERSION := $(shell cat .golangci.version)

# Directories containing independent Go modules.
MODULE_DIRS = .

.PHONY: all
all: lint test-with-redis

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  all            - Run linting and tests with Redis"
	@echo "  test           - Run tests (requires Redis to be running)"
	@echo "  test-with-redis - Start Redis, run tests, then cleanup"
	@echo "  redis          - Start Redis service"
	@echo "  redis-stop     - Stop Redis service"
	@echo "  lint           - Run linter"
	@echo "  clean          - Remove binary files"

.PHONY: clean
clean:
	@rm -rf $(GOBIN)

.PHONY: test
test:
	cd tests && go test -v

.PHONY: redis
redis:
	@echo "🚀 Starting Redis service..."
	@docker-compose up -d
	@echo "⏳ Waiting for Redis service to be ready..."
	@timeout=30; \
	counter=0; \
	until docker-compose exec redis redis-cli ping 2>/dev/null | grep -q PONG; do \
		sleep 1; \
		counter=$$((counter + 1)); \
		if [ $$counter -gt $$timeout ]; then \
			echo "❌ Timeout waiting for Redis service"; \
			docker-compose down; \
			exit 1; \
		fi; \
	done
	@echo "✅ Redis service is ready"

.PHONY: redis-stop
redis-stop:
	@echo "🧹 Stopping Redis service..."
	@docker-compose down

.PHONY: test-with-redis
test-with-redis: redis
	@echo "🧪 Running tests..."
	@$(MAKE) test
	@echo "🧹 Cleaning up test environment..."
	@docker-compose down
	@echo "✅ Tests completed!"

.PHONY: lint
lint: golangci-lint tidy-lint

# Install golangci-lint with the required version in GOBIN if it is not already installed.
.PHONY: install-golangci-lint
install-golangci-lint:
    ifneq ($(GOLANGCI_LINT_VERSION),$(REQUIRED_GOLANGCI_LINT_VERSION))
		@echo "[lint] installing golangci-lint v$(REQUIRED_GOLANGCI_LINT_VERSION) since current version is \"$(GOLANGCI_LINT_VERSION)\""
		@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) v$(REQUIRED_GOLANGCI_LINT_VERSION)
    endif

.PHONY: golangci-lint
golangci-lint: install-golangci-lint
	@echo "[lint] $(shell $(GOBIN)/golangci-lint version)"
	@$(foreach mod,$(MODULE_DIRS), \
		(cd $(mod) && \
		echo "[lint] golangci-lint: $(mod)" && \
		$(GOBIN)/golangci-lint run --path-prefix $(mod)) &&) true

.PHONY: tidy-lint
tidy-lint:
	@$(foreach mod,$(MODULE_DIRS), \
		(cd $(mod) && \
		echo "[lint] mod tidy: $(mod)" && \
		go mod tidy && \
		git diff --exit-code -- go.mod go.sum) &&) true
