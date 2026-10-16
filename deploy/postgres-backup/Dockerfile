FROM golang:1.26-alpine3.23 AS worker-build
WORKDIR /src
COPY cmd/backup-worker/main.go .
RUN CGO_ENABLED=0 GO111MODULE=off go build -trimpath -o /out/converge-backup-worker main.go

FROM postgres:16-alpine3.23
RUN apk add --no-cache pgbackrest=2.57.0-r0
RUN install -d -o postgres -g postgres -m 0700 /var/lib/converge-backup && install -d /etc/pgbackrest
COPY --from=worker-build /out/converge-backup-worker /usr/local/bin/converge-backup-worker
