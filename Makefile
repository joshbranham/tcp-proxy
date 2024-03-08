.PHONY: build, lint, test

build: $(wildcard **/*.go)
	go mod tidy
	go build -o out/ ./...

lint:
	go fmt ./...
	golangci-lint run

test:
	go test ./...
