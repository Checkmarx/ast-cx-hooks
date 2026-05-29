-include .env

.PHONY: lint
lint:
	golangci-lint run

.PHONY: build
build:
	go build ./...

.PHONY: test
test:
	go test ./... -coverprofile cover.out
