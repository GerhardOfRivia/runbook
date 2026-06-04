.PHONY: all build test clean

all: build

build:
	@echo "Building runbook..."
	go build -o runbook ./src
	@echo "Build successful! Run with: ./runbook <file_name>.shbn"

test:
	@echo "Running tests..."
	go test -v ./...

clean:
	@echo "Cleaning up..."
	rm -f runbook
