# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
BUILD_DIR=./build
BINARY_NAME=$(BUILD_DIR)/charon
BINARY_UNIX=$(BINARY_NAME)_unix

build: 
		$(GOBUILD) -trimpath -o $(BINARY_NAME) -v ./cmd
.PHONY: build

test: 
		$(GOTEST) -v ./...
.PHONY: test

clean: 
		$(GOCLEAN) ./...
		rm -rf $(BUILD_DIR)
.PHONY: clean


# Cross compilation
build-linux:
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -trimpath -o $(BINARY_NAME) -v ./cmd
.PHONY: build-linux
