# WAL backup deployment

The original Converge PostgreSQL 16 cluster uses `docker-compose.pitr.yml` with `docker-compose.pitr-live.yml`. These files keep the existing `converge_pgdata` volume and add pgBackRest archiving plus a backup worker. Silo has its own Compose project and data volume in `docker-compose.silo-live.yml`. The included Silo setup is local; move Silo and its volume to another host for protection against loss of the database host.

Create a private environment file from `live.env.example`, a private pgBackRest configuration from `pgbackrest.conf.example`, and a TLS certificate matching the Silo endpoint. Do not commit private files or lose the repository encryption passphrase. The local installation keeps these files under ignored `deploy/postgres-backup/live-private/`.

From the repository root, start storage first:

```sh
docker compose -p converge-silo --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.silo-live.yml up -d silo-live silo-live-init
```

Then start the original database and worker, supplying `.env` and the same private environment file:

```sh
docker compose -p converge --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml up -d db backup-worker
```

For this local installation, ignored `.env` selects those two PostgreSQL Compose files, so `docker compose up -d db backup-worker` keeps WAL enabled. When creating a fresh environment, set `COMPOSE_FILE=docker-compose.pitr.yml:docker-compose.pitr-live.yml` and `COMPOSE_PROJECT_NAME=converge` in `.env`, or use the explicit command above. Set the API's `PITR_WORKER_URL` and matching `PITR_WORKER_TOKEN` in its private environment. Never start the old `docker-compose.yml` database service against the live volume while the WAL-enabled service is running.

Check that PostgreSQL reports `archive_mode=on`, the worker is healthy, `pgbackrest info` lists a full backup, and `pg_stat_archiver.last_archived_wal` advances. Prove recovery by restoring to an isolated preview volume before relying on a selected time. The web preview itself does not replace live data.

When the app and database are running, Management > Backups can **Apply this recovery to live data** after a successful selected-time preview. The worker first replays WAL in its isolated database and exports that recovered state. After the admin types the exact confirmation, the API verifies the archive, makes a safety backup, pauses app requests, and replaces live data with the previewed state in one PostgreSQL transaction. The previous physical production volume is not swapped by this button. Later changes are lost from live data but retained in the safety backup. The button needs a working API, database, worker, and `BACKUP_DIR`; if the database is lost, use the separate physical outage recovery procedure below.

## Recover a lost database to the latest archived WAL

This procedure uses a **physical** pgBackRest base backup, then PostgreSQL replays every available archived WAL segment when the recovered server starts. It works without the original database or API. The Management page's logical `pg_dump` archives cannot be rolled forward with WAL. Recovery can include only transactions whose WAL reached Silo before the incident; transactions in an unarchived segment on a lost host are unavailable. The local Silo example is on the same host, so it does not protect against loss of that entire host.

Use a new Compose project name for each attempt. Its `pitrdata` volume must be empty. Preserve the damaged original volume for investigation; never run `down -v` against the live `converge` project. The private pgBackRest config, repository passphrase, Silo certificate, and Silo network must be available on the recovery host.

```sh
docker compose -p converge-recovery-1 --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml --profile pitr-restore run --rm --no-deps pitr-recover-latest
docker compose -p converge-recovery-1 --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml --profile pitr-restore up -d --no-deps pitr-db
docker compose -p converge-recovery-1 --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml logs --tail=100 pitr-db
```

The recovered server is isolated on `127.0.0.1:5434`. Check that PostgreSQL reports recovery complete, then verify important tables and records there before any operator-controlled application cutover. If restore or startup fails, leave the original volume untouched, inspect missing backup/WAL or access errors, and retry with a *new* project name. Do not run `pitr-recover-latest` again on the same volume.

### Cut over after validation

The recovery project above creates volume `converge-recovery-1_pitrdata`. Replace that name below if a different project was used. Stop the Converge API and any other application writers before cutover. Stop the isolated `pitr-db` so only one PostgreSQL server can access the recovered volume. Record the original volume name with `docker inspect postgres-safe` and preserve it for rollback.

`docker-compose.pitr-cutover.yml` changes the live project's `pgdata` reference to the **existing** recovered volume. It will fail if that volume does not exist. Review the resolved configuration before changing the running services:

```sh
export RECOVERED_PGDATA_VOLUME=converge-recovery-1_pitrdata
docker volume inspect "$RECOVERED_PGDATA_VOLUME"
docker compose -p converge-recovery-1 --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml stop pitr-db
docker compose -p converge --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml -f docker-compose.pitr-cutover.yml config --quiet
docker compose -p converge --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml stop backup-worker db
docker compose -p converge --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml -f docker-compose.pitr-cutover.yml up -d db
docker inspect postgres-safe --format '{{range .Mounts}}{{if eq .Destination "/var/lib/postgresql/data"}}{{.Name}}{{end}}{{end}}'
```

The inspected mount must equal `RECOVERED_PGDATA_VOLUME`. Verify representative records and that `archive_mode=on` on the newly started database before reopening application writes. Then make a new full backup on its new PostgreSQL timeline and restart the worker:

```sh
docker compose -p converge --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml -f docker-compose.pitr-cutover.yml exec -u postgres db pgbackrest --stanza=converge --type=full backup
docker compose -p converge --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml -f docker-compose.pitr-cutover.yml up -d backup-worker
```

If the new full backup fails, keep the API stopped and resolve the archive or backup error before reopening writes. Keep using the cutover overlay and `RECOVERED_PGDATA_VOLUME` for every later `docker compose up`; otherwise Compose will point `db` back at the old `converge_pgdata`. For this local installation, update the ignored `.env` `COMPOSE_FILE` to include `docker-compose.pitr-cutover.yml` and set `RECOVERED_PGDATA_VOLUME` there after the cutover is verified. If validation fails before reopening writes, stop the new `db`, use the original two Compose files to restart the untouched original volume, and investigate separately. Once application writes begin on the recovered database, returning to the old volume would discard those new writes; treat a later reversal as another recovery operation.

The original Silo archive is separate from both database volumes. Restoring does not replace its WAL. A cutover creates a new PostgreSQL timeline in the same pgBackRest repository; retention will eventually expire old backups and their WAL, so preserve a separate repository snapshot if the incident history must be kept longer than the configured retention period.

The existing `pitr-restore` service is for recovery to a specifically selected time; `pitr-recover-latest` is the outage procedure that replays to the end of the available archive. WAL contains PostgreSQL changes, not a human-readable list of actions. The web audit timeline is useful for previewing recent app changes while the live database is available, but is not needed for this outage restore.

The repository template retains seven full backups; pgBackRest expires old WAL with expired backups. PostgreSQL retains WAL locally while storage is unavailable, and `max_wal_size` is not a hard cap. Monitor archive failures and free space on both hosts. Keep the cold rollback volume and verified logical dump until an off-host restore drill succeeds.
