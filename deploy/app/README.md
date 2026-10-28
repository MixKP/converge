# Deployment notes

The reusable files are `docker-compose.app.yml` (frontend, API, PostgreSQL and backup worker) and `docker-compose.silo.yml` (Silo storage). `docker-compose.local.yml` joins them on one computer. Use `MODE=remote` for an app host that connects to a separate Silo server. The files do not need editing for each server; set environment values instead.

## Configuration

- `.env.app`: chosen initial admin login; generated database/JWT/worker/encryption secrets; Silo hostname/port/credentials; app bind address and port.
- `.env.silo`: reachable backup hostname, Silo credentials, bind address and ports.
- Local mode only needs `.env.app`. The local storage endpoint stays on Docker's internal `silo:9000`, the storage API is not published to the host. The local dashboard defaults to port 9011 so it can coexist with older Silo installations on port 9001.
- These files are private and ignored. No automatic seed/demo data or admin-password reset occurs.
- `make setup` copies service secrets from an existing `.env.docker` when creating `.env.app`, but deliberately leaves the admin password blank. The old file is preserved; it is no longer used by the new commands. Old root `.env`, `backend/.env`, private pgBackRest files and Compose files are preserved for existing installations.
- Use the same mode/environment files for startup and daily commands. For isolated installations or tests, `APP_PROJECT`, `SILO_PROJECT`, `APP_ENV` and `SILO_ENV` can override the default project/file names.

## Existing data

The Docker app project remains `converge-app`, so installations from the immediately preceding Docker setup keep the same named database/Silo volumes. Copy the original service credentials, encryption passphrase and any custom ports into the new environment file before starting. An existing admin continues using its existing password. Configuration generation does not rotate credentials or re-encrypt repository objects.

The earlier host-run `converge` and `converge-silo` projects use different volumes. The new standalone backup server uses `converge-storage`. They are not automatically imported or deleted. Keep the old deployment stopped during a deliberate migration, preserve its original environment/encryption keys and use a validated backup/restore. Never attach two PostgreSQL servers to the same data volume. The operator recovery/cutover procedures remain in [the recovery runbook](../postgres-backup/README.md).

## TLS and remote connectivity

Silo generates a self-signed certificate for `SILO_HOST` on first startup and retains its private key in a Docker volume. Only its public certificate is exported. Import it on the app host with `make trust-silo MODE=remote`; pgBackRest verifies certificate trust and hostname. No private key or absolute host path is shared with the API. The API communicates with its worker over the internal app network.

Use a stable reachable DNS name or IP before first Silo startup. The certificate is valid for one year. An expired certificate or hostname mismatch prevents certificate initialization. Do not disable verification to bypass the error.

For renewal, stop the app to pause writes/backups and stop Silo. Preserve both data volumes and the environment files. Use the Silo certificate-init image and its `silocerts` volume to install a new matching certificate/key, then restart Silo, export the public certificate again and reimport it on the app host before restarting the app. Existing backup objects and the encryption passphrase must stay unchanged. This is an operator maintenance task; setup never silently replaces an established certificate or key.

Only private-network access is intended by the bind defaults. A public app requires an HTTPS reverse proxy/domain; the frontend proxies `/api` on the same origin. Allow Silo's storage port only from app hosts, and restrict its dashboard to operators. In local mode both hosts share one physical computer, so a computer failure can still lose app and backup volumes together.

`make down` and `make silo-down` preserve volumes. Do not use Docker's `down -v` on an installation you need to retain.
