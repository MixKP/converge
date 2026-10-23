.PHONY: build test test-integration test-http-integration test-recovery-http-integration lint clean swag seed run

PG_BIN ?= $(firstword $(wildcard /opt/homebrew/opt/postgresql@16/bin /usr/local/opt/postgresql@16/bin))

run: swag
	go run cmd/api/main.go

swag:
	go run github.com/swaggo/swag/cmd/swag@v1.16.6 init --parseInternal -g cmd/api/main.go -o ./docs

build: swag
	mkdir -p bin
	go build -o bin/api ./cmd/api/

test:
	go test ./... -count=1

test-integration: test-http-integration
	go test -tags=integration ./internal/adapter/db/ -v -count=1
	@docker image inspect converge-postgres-backup:local >/dev/null || { echo 'Build converge-postgres-backup:local first'; exit 1; }
	@set -eu; workdir=$$(mktemp -d); trap 'rm -rf "$$workdir"' EXIT; \
	  GOOS=linux GOARCH=$$(docker image inspect converge-postgres-backup:local --format '{{.Architecture}}') CGO_ENABLED=0 go build -o "$$workdir/worker" ./cmd/backup-worker; \
	  printf 'FROM converge-postgres-backup:local\nCOPY worker /usr/local/bin/converge-backup-worker\n' > "$$workdir/Dockerfile"; \
	  docker build -t converge-postgres-backup:integration "$$workdir"; \
	  PATH="$(if $(PG_BIN),$(PG_BIN):)$$PATH" CONVERGE_PHYSICAL_DRILL=1 go test -tags=integration ./cmd/backup-worker/ -run '^TestPhysicalBackupWALPreview$$' -v -count=1

test-http-integration: test-recovery-http-integration
	go test -tags=integration ./cmd/api/ -run '^TestApplicationHTTPWithPostgres$$' -v -count=1

test-recovery-http-integration:
	PATH="$(if $(PG_BIN),$(PG_BIN):)$$PATH" go test -tags=integration ./internal/adapter/web/ -run '^TestRecoveryHTTPWithPostgres$$' -v -count=1

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

seed:
	go run ./cmd/seed/
