.PHONY: run build tidy lint

run:
	go run ./cmd/server/

build:
	go build -o bin/ayws-compute ./cmd/server/

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

dev:
	air -c .air.toml
