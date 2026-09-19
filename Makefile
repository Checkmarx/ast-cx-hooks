-include .env

.DEFAULT_GOAL := build

# Override when comparing against a different integration branch.
BASE_REV ?= origin/master

.PHONY: fmt-check lint lint-full build test
fmt-check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }

lint: fmt-check
	git rev-parse --verify $(BASE_REV)
	golangci-lint run --new-from-rev=$(BASE_REV)

# Report the existing lint backlog separately from the incremental PR gate.
lint-full:
	golangci-lint run

build:
	go build ./...
	go vet ./...

test:
	go test ./... -race -count=1 -coverprofile cover.out
