.PHONY: build test test-integration lint clean

build:
	go build -o bin/api ./cmd/api/

test:
	go test ./... -count=1

test-integration:
	go test -tags=integration ./internal/adapter/db/ -v -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/
