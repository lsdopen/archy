.PHONY: test lint build coverage clean

test:
	go test -v -race ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run

build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/webhook ./cmd/webhook

clean:
	rm -rf bin/ coverage.out coverage.html

all: lint test build