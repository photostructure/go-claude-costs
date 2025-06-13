# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=claude-costs
BINARY_UNIX=$(BINARY_NAME)_unix

.PHONY: all build clean test coverage deps lint fmt help

all: fmt test build

## Build the binary
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/claude-costs

## Clean build files
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

## Run tests
test:
	$(GOTEST) -v ./...

## Run tests with coverage
coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

## Run tests with race detection
race:
	$(GOTEST) -race -short ./...

## Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

## Format Go code
fmt:
	$(GOCMD) fmt ./...
	$(GOCMD) mod tidy

## Check formatting (CI-friendly)
fmt-check:
	@test -z "$$($(GOCMD) fmt -l ./...)" || (echo "Code not formatted. Run 'make fmt'" && exit 1)

## Run linter (requires golangci-lint)
lint:
	golangci-lint run

## Build for multiple platforms
build-all:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v ./cmd/claude-costs
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME).exe -v ./cmd/claude-costs
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)_darwin -v ./cmd/claude-costs

## Install the binary
install:
	$(GOCMD) install ./cmd/claude-costs

## Run the application
run:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/claude-costs
	./$(BINARY_NAME)

## Show help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) { \
			helpMessage = match($$2, /^[A-Z].*/); \
			if (helpMessage) { \
				printf "\033[36m%-20s\033[0m %s\n", $$1, $$2 \
			} \
		} \
	}' $(MAKEFILE_LIST)