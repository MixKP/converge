.DEFAULT_GOAL := help

MODE ?= local
APP_ENV ?= .env.app
SILO_ENV ?= .env.silo
APP_PROJECT ?= converge-app
SILO_PROJECT ?= converge-storage

ifeq ($(MODE),local)
APP_FILES := -f docker-compose.app.yml -f docker-compose.silo.yml -f docker-compose.local.yml
else ifeq ($(MODE),remote)
APP_FILES := -f docker-compose.app.yml
else
$(error MODE must be local or remote)
endif
APP := docker compose -p $(APP_PROJECT) --env-file "$(APP_ENV)" $(APP_FILES)
SILO := docker compose -p $(SILO_PROJECT) --env-file "$(SILO_ENV)" -f docker-compose.silo.yml

.PHONY: demo-data help setup setup-silo config up status logs stop down silo-config silo-up silo-status silo-logs silo-stop silo-down silo-export trust-silo

help:
	@printf '%s\n' \
	  'Local: make setup, choose ADMIN_PASSWORD in .env.app, then make up' \
	  'App with remote Silo: use MODE=remote with up/status/logs/stop/down/config' \
	  'Backup server: make setup-silo, edit .env.silo, then make silo-up' \
	  'Certificate transfer: make silo-export on backup host; make trust-silo MODE=remote on app host' \
	  'Mock data: make demo-data CONFIRM=RESET_DEMO_DATA (replaces application data)' \
	  'Daily: make status | make logs | make stop | make down (volumes are preserved)'

setup:
	@set -eu; \
	  if test -e "$(APP_ENV)"; then \
	    echo 'Existing $(APP_ENV) preserved.'; \
	  else \
	    command -v openssl >/dev/null || { echo 'OpenSSL is required to generate service secrets.'; exit 1; }; \
	    umask 077; stage=$$(mktemp "$(APP_ENV).XXXXXX"); \
	    trap 'rm -f "$$stage" "$$stage.migrated"' EXIT HUP INT TERM; \
	    sed -e "s/GENERATE_DATABASE_PASSWORD/$$(openssl rand -hex 32)/" \
	      -e "s/GENERATE_JWT_SECRET/$$(openssl rand -hex 32)/" \
	      -e "s/GENERATE_SILO_PASSWORD/$$(openssl rand -hex 32)/" \
	      -e "s/GENERATE_CIPHER_PASS/$$(openssl rand -hex 32)/" \
	      -e "s/GENERATE_WORKER_TOKEN/$$(openssl rand -hex 32)/" \
	      .env.app.example > "$$stage"; \
	    if test -f .env.docker; then \
	      awk -F= 'NR==FNR {if ($$1 ~ /^(ADMIN_USERNAME|POSTGRES_PASSWORD|JWT_SECRET|PITR_WORKER_TOKEN|PITR_CIPHER_PASS|SILO_ROOT_USER|SILO_ROOT_PASSWORD|APP_PORT|SILO_CONSOLE_PORT)$$/) saved[$$1]=substr($$0,index($$0,"=")+1); next} {if ($$1 in saved) print $$1 "=" saved[$$1]; else print}' .env.docker "$$stage" > "$$stage.migrated"; \
	      mv "$$stage.migrated" "$$stage"; \
	    fi; \
	    ln "$$stage" "$(APP_ENV)"; \
	    echo 'Created $(APP_ENV). Set ADMIN_USERNAME and your own ADMIN_PASSWORD before startup. Existing accounts are never reset.'; \
	  fi

setup-silo:
	@set -eu; \
	  if test -e "$(SILO_ENV)"; then \
	    echo 'Existing $(SILO_ENV) preserved.'; \
	  else \
	    command -v openssl >/dev/null || { echo 'OpenSSL is required to generate service secrets.'; exit 1; }; \
	    umask 077; stage=$$(mktemp "$(SILO_ENV).XXXXXX"); \
	    trap 'rm -f "$$stage"' EXIT HUP INT TERM; \
	    sed -e "s/GENERATE_SILO_PASSWORD/$$(openssl rand -hex 32)/" .env.silo.example > "$$stage"; \
	    ln "$$stage" "$(SILO_ENV)"; \
	    echo 'Created $(SILO_ENV). Set SILO_HOST and SILO_BIND_ADDRESS before startup.'; \
	  fi

config: setup
	@$(APP) config --quiet

up: config
	@docker info >/dev/null 2>&1 || { echo 'Start Docker, then retry make up.'; exit 1; }
	$(APP) up -d --build --wait --wait-timeout 300
	@printf '%s\n' 'App started. Default URL: http://localhost:8088 (APP_PORT in $(APP_ENV)).' 'Login uses your chosen admin credentials; existing passwords are preserved.'

status:
	$(APP) ps -a

logs:
	$(APP) logs --tail=100 -f

stop:
	$(APP) stop

down:
	$(APP) down

silo-config: setup-silo
	@$(SILO) config --quiet

silo-up: silo-config
	$(SILO) up -d --build --wait --wait-timeout 120

silo-status:
	$(SILO) ps -a

silo-logs:
	$(SILO) logs --tail=100 -f

silo-stop:
	$(SILO) stop

silo-down:
	$(SILO) down

silo-export:
	@set -eu; stage=$$(mktemp silo-public.crt.XXXXXX); \
	  trap 'rm -f "$$stage"' EXIT HUP INT TERM; \
	  $(SILO) run --rm --no-deps -T --entrypoint cat silo-certs-init /certs/public.crt > "$$stage"; \
	  test -s "$$stage"; mv "$$stage" silo-public.crt; \
	  echo 'Exported silo-public.crt. Transfer this public certificate to the app host using a trusted channel.'

trust-silo:
	@test -s silo-public.crt || { echo 'Copy silo-public.crt from the backup host to this directory first.'; exit 1; }
	$(APP) run --rm --build --no-deps -T --entrypoint /bin/sh backup-config-init -ec \
	  'umask 077; stage=$$(mktemp /config/.ca.XXXXXX); trap '\''rm -f "$$stage"'\'' EXIT HUP INT TERM; cat > "$$stage"; openssl x509 -in "$$stage" -checkend 0 -noout; chown postgres:postgres "$$stage"; chmod 640 "$$stage"; mv "$$stage" /config/silo-ca.crt' < silo-public.crt

demo-data:
	@test "$(CONFIRM)" = RESET_DEMO_DATA || { echo 'This replaces application data. For a disposable setup, run make demo-data CONFIRM=RESET_DEMO_DATA'; exit 1; }
	@$(APP) config --quiet
	$(APP) stop frontend backend
	$(APP) run --rm --build --no-deps -e SEED_DEMO_USERS=true --entrypoint /usr/local/bin/seed admin-init
	$(APP) up -d --no-deps --wait --wait-timeout 120 backend frontend
