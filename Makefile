BINARY_NAME=bin/google-jules-mcp

.PHONY: all build test lint clean run

all: lint test build

build:
	@mkdir -p bin
	go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/google-jules-mcp

test:
	go test -v -race -cover ./...

lint:
	go vet ./...

run: build
	./$(BINARY_NAME)

clean:
	rm -rf bin/
