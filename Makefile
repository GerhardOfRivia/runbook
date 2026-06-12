SEMVER ?= 1.0.1
VERSION := $(SEMVER)-dev
LDFLAGS = -ldflags "-X main.Version=$(VERSION)"

.PHONY: all build test clean

all: build

build:
	@echo "Building runbook with version $(VERSION)..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o runbook ./src
	@echo "Build successful! Run with: ./runbook <file_name>.shbn"

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up..."
	rm -f runbook
