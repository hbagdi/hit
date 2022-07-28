
all: lint test

test:
	go test -race ./...

lint:
	golangci-lint run ./...

