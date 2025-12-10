.PHONY: build build-amd64 build-arm64 build-multiarch build-multiarch-push clean

BINARY_NAME=webhook
BUILD_DIR=bin
IMAGE_TAG?=archy-webhook:latest

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/webhook

build-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-amd64 ./cmd/webhook

build-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-arm64 ./cmd/webhook

build-multiarch:
	docker buildx build --platform linux/amd64,linux/arm64 -t archy-webhook:latest .

build-multiarch-push:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE_TAG) --push .

clean:
	rm -rf $(BUILD_DIR)
