.PHONY: build test test-integration lint clean swag seed run

run: swag
	go run cmd/api/main.go

swag:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go -o ./docs

build: swag
	mkdir -p bin
	go build -o bin/api ./cmd/api/

test:
	go test ./... -count=1

test-integration:
	go test -tags=integration ./internal/adapter/db/ -v -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

seed:
	go run ./cmd/seed/
