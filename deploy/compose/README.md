# OctoLink Docker Compose — Local E2E Quick Start

This directory contains the Docker Compose stack for running OctoLink locally, which is the
recommended way to run end-to-end (e2e) acceptance tests.

## Prerequisites

- Docker 24+ with the Compose plugin (`docker compose version`)
- At least 4 GB of free RAM

## 1. Bootstrap env files

Each service needs a `.env.<svc>` file. On first run, `run.sh` automatically copies each
`.env.<svc>.example` template to `.env.<svc>` if the target file does not yet exist.

```bash
cd deploy/compose
./run.sh
```

> **Security note:** The default dev credentials (`change_me_in_production`) in the example
> templates are intentionally weak — suitable for local QA only. Never use them in production.

To customise, edit the generated `.env.*` files **before** starting the stack. The real `.env.*`
files are listed in `.gitignore` and will never be committed.

## 2. Start the stack

```bash
# First run — bootstraps env files then starts all services:
./run.sh

# Subsequent runs — skip the warning and go straight to compose up:
COMPOSE_PROFILES=nats,controller,cwmp,mqtt,stomp,ws,adapter,frontend,portainer docker compose up -d
```

## 3. Verify the stack is healthy

```bash
docker compose ps
```

Expected state: all containers in `running` or `healthy`.

## 4. Smoke-test endpoints

| Service      | URL                         | Notes                     |
|--------------|-----------------------------|---------------------------|
| Frontend     | http://localhost:3000       | Next.js UI                |
| Controller   | http://localhost:8000       | REST API                  |
| SocketIO     | http://localhost:5000       | Real-time events          |
| MQTT         | tcp://localhost:1883        | USP MTP                   |
| WebSocket    | ws://localhost:8080         | USP MTP                   |
| ACS (CWMP)   | http://localhost:9292       | TR-069 ACS                |
| File server  | http://localhost:8004       | Firmware/image files      |
| Portainer    | https://localhost:9443      | Container management UI   |

Quick HTTP checks:

```bash
curl -s -o /dev/null -w "%{http_code}" http://localhost:3000   # expect 200
curl -s -o /dev/null -w "%{http_code}" http://localhost:8000   # expect 200 or 404
curl -s -o /dev/null -w "%{http_code}" http://localhost:5000   # expect 200
```

## 5. Stop the stack

```bash
./stop.sh
```

## 6. E2E test cases supported

Once the stack is up, QA can run the following acceptance scenarios against the local stack:

- **E2E-SIM-001 ~ 005**: Agent simulation flows via MQTT / WebSocket / STOMP
- **RBAC**: Role-based access control on the controller API
- **Notification room isolation**: SocketIO room scoping per device/tenant

## Troubleshooting

### `docker compose config` fails with "env_file not found"

Run `./run.sh` once — it will auto-generate missing `.env.*` files from the bundled `.env.*.example`
templates. After that, re-run `docker compose config` to confirm zero errors.

### macOS: `go test ./...` fails with `undefined: unix.TCPInfo`

This is a **Linux-only system call**. The ACS and MTP adapter packages use Linux-specific socket
options that cannot be compiled on macOS. This is **expected** and is NOT a product bug.
Acceptance testing must be done against Docker images (which run Linux containers), not by running
`go test` directly on the macOS host.

### NATS TLS errors on startup

The NATS config in `./nats_config/` includes self-signed certificates for the internal overlay
network. These are generated during the first `up` and are not present in Git. If you see
certificate errors, ensure the `nats_config` directory is writable and re-run `./run.sh`.
