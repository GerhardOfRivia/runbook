VERSION ?= 1.0.0
TAG_VERSION := $(VERSION)-dev
LDFLAGS = -ldflags "-X main.Version=$(TAG_VERSION)"

.PHONY: all build test clean

all: build

build:
	@echo "Building runbook with version $(TAG_VERSION)..."
	go build $(LDFLAGS) -o runbook ./src
	@echo "Build successful! Run with: ./runbook <file_name>.shbn"

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up..."
	rm -f runbook
