FROM golang:1.26-alpine3.23 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go run github.com/swaggo/swag/cmd/swag@v1.16.6 init --parseInternal -g cmd/api/main.go -o ./docs
RUN CGO_ENABLED=0 go build -trimpath -o /out/api ./cmd/api && \
    CGO_ENABLED=0 go build -trimpath -o /out/bootstrap ./cmd/bootstrap && \
    CGO_ENABLED=0 go build -trimpath -o /out/seed ./cmd/seed

FROM postgres:16-alpine3.23
RUN addgroup -g 10001 converge && adduser -D -u 10001 -G converge converge && \
    mkdir -p /var/lib/converge/backups && chown converge:converge /var/lib/converge/backups
WORKDIR /app
COPY --from=build /out/api /out/bootstrap /out/seed /usr/local/bin/
USER converge
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
