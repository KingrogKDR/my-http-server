
APP_NAME := my-http-server
BUILD_DIR := bin
SOURCE_DIR := server
MAIN_FILE := $(SOURCE_DIR)/server.go

.PHONY: build run test clean

build:
	@echo "Building the app..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "Build complete. Executable is located at $(BUILD_DIR)/$(APP_NAME)"

run: build
	@echo "Running the app..."
	@./$(BUILD_DIR)/$(APP_NAME)
	@echo "App terminated."

test: 
	@echo "Test start..."
	@go test -v ./...
	@echo "Test complete."

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."
