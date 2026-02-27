.PHONY: build run clean test install

BINARY_NAME=lume
BUILD_DIR=.
CMD_PATH=./cmd/lume/...

# 默认构建目标
all: build

# 构建项目
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# 运行项目
run:
	go run $(CMD_PATH)

# 清理构建产物
clean:
	@echo "Cleaning..."
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Clean complete"

# 运行测试
test:
	go test -v ./...

# 安装依赖
deps:
	go mod tidy

# 安装到系统
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete"

# 卸载
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstall complete"
