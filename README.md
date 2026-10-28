# Converge

Requires Docker Compose, Make and OpenSSL. Run commands from the repository root.

## One computer

```sh
make setup
```

Edit `.env.app`: choose `ADMIN_USERNAME` and **your own `ADMIN_PASSWORD`**. Single-quote passwords containing `$` or `#`. Then:

```sh
make up
```

Open **http://localhost:8088**. Silo's dashboard is **https://localhost:9011**; its login is `SILO_ROOT_USER` / `SILO_ROOT_PASSWORD` in `.env.app`. The local certificate is self-signed.

## Optional mock data

After `make up`, run `make demo-data CONFIRM=RESET_DEMO_DATA` on a disposable database. **This replaces application data.** It creates sample teachers, branches, subjects and bookings, plus `student1`–`student4` and `parent1` with password `password`. Your admin password stays unchanged. Normal startup never adds mock data.

## Separate app and backup servers

On the **backup server**:

```sh
make setup-silo
```

Edit `.env.silo`: set `SILO_HOST` to its reachable DNS name or IP, and `SILO_BIND_ADDRESS` to its private network IP. Then:

```sh
make silo-up
make silo-export
```

Transfer the exported **`silo-public.crt`** to the app server's repository root using a trusted channel.

On the **app server**:

```sh
make setup
```

Edit `.env.app`: choose the admin login, copy `SILO_ROOT_USER` and `SILO_ROOT_PASSWORD` from the backup server, and set its `SILO_HOST` / `SILO_PORT`. Then:

```sh
make trust-silo MODE=remote
make up MODE=remote
```

Allow the app server to reach Silo's storage port (default 9000). Restrict dashboard access (9001) to operators. Set `APP_BIND_ADDRESS` to the app server's private IP when exposing the frontend through your HTTPS reverse proxy. Compose does not create DNS or firewall rules.

## Daily commands

`make status`, `make logs`, `make stop`, `make down`. Add `MODE=remote` on the separate app server. Use `make silo-status`, `make silo-logs`, `make silo-stop`, `make silo-down` on the backup server. Stop/down preserve volumes.

Setup generates service secrets once and leaves the admin password blank for you. Repeating setup preserves existing files; changing the environment password does not reset an existing admin. Keep your environment files safe, especially `PITR_CIPHER_PASS`, which is needed to read backups.

Docker manages database/storage volumes, certificates, backup configuration and the API's safety-backup directory. No `BACKUP_DIR` or certificate/config paths need entering. Physical backups and archived WAL live in Silo; Apply still makes a logical safety backup internally.

This uses separate volumes from the older host-run stack; existing business data is not automatically transferred. See [deployment notes](deploy/app/README.md) for migration, certificate renewal and optional [host-run development](deploy/app/host-development.md). Root `.env` and `backend/.env` are only for that older development workflow, not these Docker commands.
