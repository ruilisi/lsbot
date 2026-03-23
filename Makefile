VERSION := 2.0.6
BUILD := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
PROJECTNAME := lsbot
GOBASE := $(shell pwd)
GOBIN := $(GOBASE)/dist
GOARCH ?= $(shell go env GOARCH)
GOOS ?= $(shell go env GOOS)
LDFLAGS=-ldflags "-X github.com/pltanton/lsbot/internal/mcp.ServerVersion=$(VERSION) -X main.Build=$(BUILD) -w -s"
LDFLAGS_DEBUG=-ldflags "-X github.com/pltanton/lsbot/internal/mcp.ServerVersion=$(VERSION) -X main.Build=$(BUILD) -X github.com/pltanton/lsbot/internal/debug.enabled=true"
GOBUILD=go build $(LDFLAGS)
GOBUILD_DEBUG=go build $(LDFLAGS_DEBUG)

.PHONY: all build build-debug clean install uninstall test darwin-all darwin-arm64 darwin-amd64 darwin-universal linux-all linux-amd64 linux-arm64 windows-all windows-amd64 windows-arm64

# Default: build for current platform
build:
	$(GOBUILD) -o $(GOBIN)/$(PROJECTNAME) .

# Debug build with verbose logging
build-debug:
	$(GOBUILD_DEBUG) -o $(GOBIN)/$(PROJECTNAME)-debug .

# Build all platforms
all: darwin-all linux-all windows-all

darwin-all: darwin-amd64 darwin-arm64

# macOS builds
darwin-amd64:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin $(GOBUILD) -o $(GOBIN)/$(PROJECTNAME)-$(VERSION)-darwin-amd64 .

darwin-arm64:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=darwin $(GOBUILD) -o $(GOBIN)/$(PROJECTNAME)-$(VERSION)-darwin-arm64 .

darwin-universal: darwin-amd64 darwin-arm64
	lipo -create -output $(GOBIN)/$(PROJECTNAME)-$(VERSION)-darwin-universal \
		$(GOBIN)/$(PROJECTNAME)-$(VERSION)-darwin-amd64 \
		$(GOBIN)/$(PROJECTNAME)-$(VERSION)-darwin-arm64

# Linux builds
linux-amd64:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux $(GOBUILD) -o $(GOBIN)/$(PROJECTNAME)-$(VERSION)-linux-amd64 .

linux-arm64:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=linux $(GOBUILD) -o $(GOBIN)/$(PROJECTNAME)-$(VERSION)-linux-arm64 .

linux-all: linux-amd64 linux-arm64

# Windows builds
windows-amd64:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows $(GOBUILD) -o $(GOBIN)/$(PROJECTNAME)-$(VERSION)-windows-amd64.exe .

windows-arm64:
	CGO_ENABLED=0 GOARCH=arm64 GOOS=windows $(GOBUILD) -o $(GOBIN)/$(PROJECTNAME)-$(VERSION)-windows-arm64.exe .

windows-all: windows-amd64 windows-arm64

# Code signing (macOS)
codesign:
	codesign --verbose --force --deep -o runtime --sign "Developer ID Application: Suzhou Ruilisi Technology Co.,Ltd. (ACK44BB9HY)" $(GOBIN)/$(PROJECTNAME)-$(VERSION)-darwin-universal

# Install as system service
install: build
	@echo "Installing $(PROJECTNAME)..."
	$(GOBIN)/$(PROJECTNAME) service install

# Uninstall system service
uninstall:
	@echo "Uninstalling $(PROJECTNAME)..."
	$(GOBIN)/$(PROJECTNAME) service uninstall

# Start service
start:
	$(GOBIN)/$(PROJECTNAME) service start

# Stop service
stop:
	$(GOBIN)/$(PROJECTNAME) service stop

# Service status
status:
	$(GOBIN)/$(PROJECTNAME) service status

# Run tests
test:
	go test -race -count=1 ./...

# Clean build artifacts
clean:
	rm -rf $(GOBIN)
	rm -f $(PROJECTNAME)

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Run locally (for development)
run:
	go run . serve

# Show version
version:
	@echo "$(PROJECTNAME) $(VERSION) ($(BUILD))"
