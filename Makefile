# Makefile for iperf-go
# 支持多平台交叉编译

# 项目信息
BINARY_NAME=iperf-go
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go 相关配置
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# 编译标志
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 输��目录
OUTPUT_DIR=build
DIST_DIR=dist

# 支持的平台和架构
PLATFORMS=linux windows darwin
ARCHITECTURES=amd64 arm64

# 默认目标
.PHONY: all
all: clean build

# 本地编译
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) for local platform..."
	@mkdir -p $(OUTPUT_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)
	@echo "Build complete: $(OUTPUT_DIR)/$(BINARY_NAME)"

# 编译所有平台
.PHONY: build-all
build-all: clean
	@echo "Building for all platforms..."
	@$(MAKE) build-linux
	@$(MAKE) build-windows
	@$(MAKE) build-darwin
	@echo "All builds complete!"

# Linux 平台
.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(OUTPUT_DIR)/linux
	@for arch in $(ARCHITECTURES); do \
		echo "  Building linux/$$arch..."; \
		GOOS=linux GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/linux/$(BINARY_NAME)-linux-$$arch; \
	done

# Windows 平台
.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(OUTPUT_DIR)/windows
	@for arch in $(ARCHITECTURES); do \
		echo "  Building windows/$$arch..."; \
		GOOS=windows GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/windows/$(BINARY_NAME)-windows-$$arch.exe; \
	done

# macOS 平台
.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(OUTPUT_DIR)/darwin
	@for arch in $(ARCHITECTURES); do \
		echo "  Building darwin/$$arch..."; \
		GOOS=darwin GOARCH=$$arch $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/darwin/$(BINARY_NAME)-darwin-$$arch; \
	done

# 打包所有构建产物
.PHONY: package
package: build-all
	@echo "Packaging builds..."
	@mkdir -p $(DIST_DIR)
	@cd $(OUTPUT_DIR)/linux && tar czf ../../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(OUTPUT_DIR)/linux && tar czf ../../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	@cd $(OUTPUT_DIR)/windows && zip -q ../../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	@cd $(OUTPUT_DIR)/windows && zip -q ../../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-arm64.zip $(BINARY_NAME)-windows-arm64.exe
	@cd $(OUTPUT_DIR)/darwin && tar czf ../../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(OUTPUT_DIR)/darwin && tar czf ../../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@echo "Packaging complete! Files in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)

# 清理编译产物
.PHONY: clean
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(OUTPUT_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete!"

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# 下载依赖
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# 安装到本地
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(OUTPUT_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# 运行服务端（用于快速测试）
.PHONY: run-server
run-server: build
	@echo "Running server..."
	@$(OUTPUT_DIR)/$(BINARY_NAME) -s

# 运行客户端（用于快速测试）
.PHONY: run-client
run-client: build
	@echo "Running client..."
	@$(OUTPUT_DIR)/$(BINARY_NAME) -c 127.0.0.1

# 显示版本信息
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"

# 帮助信息
.PHONY: help
help:
	@echo "iperf-go Makefile 使用说明"
	@echo ""
	@echo "可用目标："
	@echo "  make build          - 编译本地平台版本"
	@echo "  make build-all      - 编译所有平台版本"
	@echo "  make build-linux    - 编译 Linux 版本（amd64, arm64）"
	@echo "  make build-windows  - 编译 Windows 版本（amd64, arm64）"
	@echo "  make build-darwin   - 编译 macOS 版本（amd64, arm64）"
	@echo "  make package        - 打包所有平台版本"
	@echo "  make clean          - 清理编译产物"
	@echo "  make test           - 运行测试"
	@echo "  make deps           - 下载并整理依赖"
	@echo "  make install        - 安���到本地 GOPATH/bin"
	@echo "  make run-server     - 运行服务端（快速测试）"
	@echo "  make run-client     - 运行客户端（快速测试）"
	@echo "  make version        - 显示版本信息"
	@echo "  make help           - 显示此帮助信息"
	@echo ""
	@echo "示例："
	@echo "  make                 # 等同于 make all"
	@echo "  make build           # 编译当前平台"
	@echo "  make build-all       # 编译所有支持的��台"
	@echo "  make package         # 编译并打包所有平台"
