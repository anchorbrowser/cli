SHELL := /bin/zsh

APP := anchorbrowser
PKG := ./...

.PHONY: generate fmt lint test test-race vulncheck build release-check

generate:
	go generate ./...

fmt:
	gofmt -w $(shell go list -f '{{.Dir}}' ./...)
	go run golang.org/x/tools/cmd/goimports@v0.31.0 -w $(shell go list -f '{{.Dir}}' ./...)

lint:
	go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8 run

test:
	go test $(PKG)

test-race:
	go test -race $(PKG)

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

build:
	go build -o bin/$(APP) ./cmd/anchorbrowser

release-check:
	go run github.com/goreleaser/goreleaser/v2@v2.8.1 check
