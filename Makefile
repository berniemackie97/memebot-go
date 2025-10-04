SHELL := /bin/bash
APP := memebot
BIN_DIR := bin

.PHONY: build run test lint
build:
	@mkdir -p 
	go build -o /paper ./cmd/paper
	go build -o /executor ./cmd/executor

run:
	go run ./cmd/paper

test:
	go test ./...

lint:
	@echo "(optional) add golangci-lint here"

run-dex:
	go run ./cmd/dexexec
