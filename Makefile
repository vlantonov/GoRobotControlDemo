.PHONY: help build test vet clean run

.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Compile the binary
	go build -o robotdemo ./cmd/robotdemo

test: ## Run all tests with race detector
	go test -race -count=1 ./...

vet: ## Run go vet on all packages
	go vet ./...

clean: ## Remove compiled binary
	rm -f robotdemo

run: ## Run the robot demo
	go run ./cmd/robotdemo
