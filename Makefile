# Makefile for building storygen for multiple platforms
# Run on Ubuntu or other Linux systems

APPNAME=storygen
MAIN_PATH=.
BUILD_DIR=build

# Default build is for current OS/Arch
.PHONY: build
build:
	@echo "Building $(APPNAME) for current platform..."
	go build -o $(BUILD_DIR)/$(APPNAME) $(MAIN_PATH)

# Clean build directory
.PHONY: clean
clean:
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)
	mkdir -p $(BUILD_DIR)

# Build for all supported platforms
.PHONY: all
all: clean windows macos linux

# Build for Windows (various architectures)
.PHONY: windows
windows:
	@echo "Building for Windows/amd64..."
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(APPNAME)_windows_amd64.exe $(MAIN_PATH)
	@echo "Building for Windows/386..."
	GOOS=windows GOARCH=386 go build -o $(BUILD_DIR)/$(APPNAME)_windows_386.exe $(MAIN_PATH)
	@echo "Building for Windows/arm64..."
	GOOS=windows GOARCH=arm64 go build -o $(BUILD_DIR)/$(APPNAME)_windows_arm64.exe $(MAIN_PATH)

# Build for MacOS (various architectures)
.PHONY: macos
macos:
	@echo "Building for MacOS/amd64..."
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(APPNAME)_darwin_amd64 $(MAIN_PATH)
	@echo "Building for MacOS/arm64..."
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(APPNAME)_darwin_arm64 $(MAIN_PATH)

# Build for Linux (various architectures)
.PHONY: linux
linux:
	@echo "Building for Linux/amd64..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(APPNAME)_linux_amd64 $(MAIN_PATH)
	@echo "Building for Linux/386..."
	GOOS=linux GOARCH=386 go build -o $(BUILD_DIR)/$(APPNAME)_linux_386 $(MAIN_PATH)
	@echo "Building for Linux/arm64..."
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(APPNAME)_linux_arm64 $(MAIN_PATH)
	@echo "Building for Linux/arm..."
	GOOS=linux GOARCH=arm go build -o $(BUILD_DIR)/$(APPNAME)_linux_arm $(MAIN_PATH)

# Create compressed archives for distribution
.PHONY: dist
dist: all
	@echo "Creating distribution archives..."
	cd $(BUILD_DIR) && tar -czvf $(APPNAME)_linux_amd64.tar.gz $(APPNAME)_linux_amd64
	cd $(BUILD_DIR) && tar -czvf $(APPNAME)_linux_386.tar.gz $(APPNAME)_linux_386
	cd $(BUILD_DIR) && tar -czvf $(APPNAME)_linux_arm64.tar.gz $(APPNAME)_linux_arm64
	cd $(BUILD_DIR) && tar -czvf $(APPNAME)_linux_arm.tar.gz $(APPNAME)_linux_arm
	cd $(BUILD_DIR) && tar -czvf $(APPNAME)_darwin_amd64.tar.gz $(APPNAME)_darwin_amd64
	cd $(BUILD_DIR) && tar -czvf $(APPNAME)_darwin_arm64.tar.gz $(APPNAME)_darwin_arm64
	cd $(BUILD_DIR) && zip $(APPNAME)_windows_amd64.zip $(APPNAME)_windows_amd64.exe
	cd $(BUILD_DIR) && zip $(APPNAME)_windows_386.zip $(APPNAME)_windows_386.exe
	cd $(BUILD_DIR) && zip $(APPNAME)_windows_arm64.zip $(APPNAME)_windows_arm64.exe

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  clean       - Clean build directory"
	@echo "  all         - Build for all supported platforms"
	@echo "  windows     - Build for Windows (386, amd64, arm64)"
	@echo "  macos       - Build for MacOS (amd64, arm64)"
	@echo "  linux       - Build for Linux (386, amd64, arm64, arm)"
	@echo "  dist        - Create distribution archives"
	@echo "  help        - Show this help message"
