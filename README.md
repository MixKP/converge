# Converge

Converge is a Vue frontend and Go API backed by PostgreSQL 16. The API and frontend run on your computer during development; Docker runs the database. Physical backups and WAL recovery are optional for ordinary app development.

## Requirements

- Docker with Docker Compose
- Go 1.26.6 (the toolchain specified in `backend/go.mod`)
- Node.js and npm compatible with `frontend/package-lock.json`
- PostgreSQL 16 client tools (`pg_dump` and `pg_restore`) if you enable Management > Backups

Run the commands below from the repository root unless a step says otherwise.

## Start a fresh local app

1. Create private environment files:

   ```sh
   cp .env.example .env
   cp backend/.env.example backend/.env
   ```

   Set `POSTGRES_PASSWORD` to the same value in both files. In `backend/.env`, set a long random `JWT_SECRET` and an `ADMIN_PASSWORD`. Keep `DB_HOST=localhost` and `DB_PORT=5432` when the API runs on your computer. The example files are templates; `.env` files are ignored by Git.

2. Start PostgreSQL:

   ```sh
   docker compose -f docker-compose.yml up -d --wait db
   ```

3. On a **new, disposable database only**, create the admin account and sample application data:

   ```sh
   cd backend
   go run ./cmd/seed
   go run ./cmd/api
   ```

   Keep the API terminal open. `go run ./cmd/seed` **truncates and recreates** several application tables. Do not run it against a database containing records you need to keep. The API runs its schema migrations when it starts, but does not create the first admin account. Changing `ADMIN_PASSWORD` later does not change an existing admin's password.

4. In another terminal, start the frontend:

   ```sh
   cd frontend
   npm ci
   npm run dev
   ```

Open [http://localhost:5173](http://localhost:5173) and sign in with `ADMIN_USERNAME` (default `admin`) and the password set before seeding. Use `localhost`: the API's default CORS origin and the frontend's default API URL both use that hostname. The API is at [http://localhost:8080](http://localhost:8080); check [http://localhost:8080/health](http://localhost:8080/health) if the frontend cannot connect.

For subsequent starts, run the database, API, and frontend commands again, **without** rerunning the seed command. `SEED_DEMO_USERS=true` in `backend/.env` adds local demo accounts when seeding; leave it false for shared databases.

### Enable logical backups in the Management page

Create a writable directory and set its absolute path as `BACKUP_DIR` in `backend/.env`. The API schedules backups while it runs; `BACKUP_INTERVAL_HOURS` and `BACKUP_RETENTION_DAYS` are optional. PostgreSQL 16 `pg_dump` and `pg_restore` must be on the API process's `PATH`. This gives you scheduled and manual logical backups. A directory on the same computer does not protect against loss of that computer; use a mounted off-host directory for durable deployment backups.

## Run tests

Run these commands from `backend/`:

- `make test` runs the unit tests.
- `make test-integration` runs the application HTTP, recovery HTTP, repository, and physical Silo/WAL integration tests, together with the unit tests. Docker must be running. It requires PostgreSQL 16 client tools and the `converge-postgres-backup:local` image.

The application HTTP test uses the production router, real services and SQL repositories. Mocked handler tests remain for validation and injected failures.

## Run the physical backup and WAL recovery stack

This is an additional setup for testing recovery from a physical base backup plus archived WAL. It runs Silo in a separate Compose project. The local example still keeps both Docker projects on one computer; deployment storage should be on another host.

1. Configure the basic `.env` and `backend/.env` files above. Set `BACKUP_DIR` too if you want the Management page's **Apply recovery** button, which makes a safety backup before replacing live data.
2. Create private configuration files:

   ```sh
   mkdir -p deploy/postgres-backup/live-private/silo-certs
   cp deploy/postgres-backup/live.env.example deploy/postgres-backup/live-private/live.env
   cp deploy/postgres-backup/pgbackrest.conf.example deploy/postgres-backup/live-private/pgbackrest-live.conf
   ```

   Replace every placeholder. The Silo root user/password must match the pgBackRest S3 key/secret; use an independent long `repo1-cipher-pass` and keep it safe. For the local Compose setup, set the pgBackRest S3 bucket to `converge-live-pitr`, endpoint to `silo-live`, and these additional options in `pgbackrest-live.conf`:

   ```ini
   repo1-storage-port=9000
   repo1-storage-ca-file=/etc/pgbackrest/silo-ca.crt
   repo1-storage-verify-tls=y
   ```
3. Give Silo a certificate valid for `silo-live`. Put `public.crt` and `private.key` in the directory named by `SILO_CERT_DIR`; set `SILO_CA_FILE` to the absolute path of `public.crt`. For a disposable local test certificate, run this in that directory:

   ```sh
   openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
     -keyout private.key -out public.crt -subj '/CN=silo-live' \
     -addext 'subjectAltName=DNS:silo-live'
   ```

4. Set absolute `SILO_CERT_DIR`, `SILO_CA_FILE`, and `PGBACKREST_CONFIG_FILE` paths in the private `live.env`. Generate a long `PITR_WORKER_TOKEN` there. Put the **same** token and `PITR_WORKER_URL=http://127.0.0.1:8099` in `backend/.env`.
5. Start Silo first, then the WAL-enabled database and worker, from the repository root. Replace the private environment path if you chose another one:

   ```sh
   docker compose -p converge-silo --env-file deploy/postgres-backup/live-private/live.env \
     -f docker-compose.silo-live.yml up -d silo-live
   docker compose -p converge-silo --env-file deploy/postgres-backup/live-private/live.env \
     -f docker-compose.silo-live.yml run --rm silo-live-init

   docker compose -p converge --env-file .env \
     --env-file deploy/postgres-backup/live-private/live.env \
     -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml \
     up -d --build db backup-worker
   ```

   Then run the API and frontend as above. Seed only if this is a new disposable database. Check the worker with `docker compose -p converge --env-file .env --env-file deploy/postgres-backup/live-private/live.env -f docker-compose.pitr.yml -f docker-compose.pitr-live.yml ps` and open Management > Backups after signing in. The first physical backup may take time to finish.

The local `.env` on one developer's machine may define `COMPOSE_FILE`; the explicit `-f` options above make the selected services unambiguous. Never start `docker-compose.yml`'s plain database against a volume being used by the WAL-enabled database. Private environment files, certificates, repository keys, and the encryption passphrase are not included in Git. Each developer needs their own test credentials or a securely provided team environment.

For recovery procedures, validation, and cutover, read [the WAL deployment runbook](deploy/postgres-backup/README.md). Do not test **Apply recovery** on data you need to keep: it replaces the live database after confirmation.
