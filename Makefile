test: lint
	go test -test -count 1 ./...

lint:
	golangci-lint run ./...

