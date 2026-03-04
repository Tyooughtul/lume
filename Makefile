.PHONY: build run clean test install

BINARY_NAME=lume
BUILD_DIR=.
CMD_PATH=./cmd/lume/...

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Default build target
all: build

# Build the project
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build -ldflags "-s -w -X github.com/Tyooughtul/lume/pkg/ui.AppVersion=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run the project
run:
	go run $(CMD_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Clean complete"

# Run tests
test:
	go test -v ./...

# Install dependencies
deps:
	go mod tidy

# Install to system
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete"

# Uninstall
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstall complete"
