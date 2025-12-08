.PHONY: all build test clean install

# Build variables
BINARY_NAME=autoengineer
VERSION=2.3.3
BUILD_DIR=dist
GO_CMD=go
GO_BUILD=$(GO_CMD) build
GO_TEST=$(GO_CMD) test
GO_CLEAN=$(GO_CMD) clean

# Build flags
LDFLAGS=-ldflags "-s -w"

all: test build

# Build for current platform
build:
	cd go && $(GO_BUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/autoengineer

# Run tests
test:
	cd go && $(GO_TEST) -v ./...

# Run tests with coverage
test-coverage:
	cd go && $(GO_TEST) -v -coverprofile=coverage.txt -covermode=atomic ./...

# Build for all platforms
build-all: clean
	mkdir -p $(BUILD_DIR)
	# Linux amd64
	cd go && GOOS=linux GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o ../$(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/autoengineer
	# Linux arm64
	cd go && GOOS=linux GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o ../$(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/autoengineer
	# macOS amd64
	cd go && GOOS=darwin GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o ../$(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/autoengineer
	# macOS arm64
	cd go && GOOS=darwin GOARCH=arm64 $(GO_BUILD) $(LDFLAGS) -o ../$(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/autoengineer
	# Windows amd64
	cd go && GOOS=windows GOARCH=amd64 $(GO_BUILD) $(LDFLAGS) -o ../$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/autoengineer

# Install binary to $HOME/.local/bin
install: build
	mkdir -p $(HOME)/.local/bin
	cp go/$(BINARY_NAME) $(HOME)/.local/bin/
	chmod +x $(HOME)/.local/bin/$(BINARY_NAME)

# Clean build artifacts
clean:
	cd go && $(GO_CLEAN)
	rm -rf $(BUILD_DIR)
	rm -f go/$(BINARY_NAME)

# Run linters (requires golangci-lint)
lint:
	cd go && golangci-lint run

# Format code
fmt:
	cd go && go fmt ./...

# Download dependencies
deps:
	cd go && go mod download
	cd go && go mod tidy

# Show help
help:
	@echo "AutoEngineer Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all          - Run tests and build"
	@echo "  build        - Build for current platform"
	@echo "  build-all    - Build for all platforms (Linux, macOS, Windows)"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  install      - Install binary to ~/.local/bin"
	@echo "  clean        - Remove build artifacts"
	@echo "  lint         - Run linters"
	@echo "  fmt          - Format code"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  help         - Show this help message"
