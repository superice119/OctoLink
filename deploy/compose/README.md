# OctoLink Docker Compose — Local Source Build Deployment

This stack now builds OctoLink services directly from the checked-out source tree.
Custom services no longer pull `oktopusp/*` images from Docker Hub; only the shared
infrastructure services (`nats`, `mongo`, `nginx`, `portainer`) still use upstream images.

## Prerequisites

- Docker Engine with the Compose plugin
- GNU Make
- OpenSSL
- At least 4 GB of free RAM

## 1. Bootstrap env files

From `deploy/compose/`, run `./run.sh` once to generate missing `.env.*` files from the
bundled examples, or copy the templates manually before starting the stack.

> The bundled `.env.*.example` files contain development-only placeholders. Never commit
> real production secrets.

## 2. Generate test TLS certificates

NATS TLS in this compose stack expects local test certificates under `./nats_config/`.
Generate them before the first startup:

```bash
cd deploy/compose
./gen-test-certs.sh
```

The script creates:

- `nats_config/rootCA.pem`
- `nats_config/cert.pem`
- `nats_config/key.pem`

These files are for local testing only and are excluded from git.

## 3. Build local images

You can build everything from the repository root with the restored top-level Makefile:

```bash
cd build
make build DOCKER_USER=octolink
```

Or let Compose build on demand:

```bash
cd deploy/compose
COMPOSE_PROFILES=nats,controller,cwmp,mqtt,stomp,ws,adapter,frontend,portainer docker compose build
```

## 4. Start the stack from local source

```bash
cd deploy/compose
COMPOSE_PROFILES=nats,controller,cwmp,mqtt,stomp,ws,adapter,frontend,portainer docker compose up --build -d
```

Each OctoLink service is configured with a local `build:` context plus `pull_policy: never`,
so Compose will use locally built images such as `octolink/controller:local` instead of
falling back to Docker Hub.

## 5. Verify the stack

```bash
docker compose ps
```

Recommended smoke checks:

- Frontend: `http://localhost:3000`
- Controller API: `http://localhost:8000`
- SocketIO: `http://localhost:5000`
- ACS: `http://localhost:9292`
- File server: `http://localhost:8004`

## 6. Stop the stack

```bash
./stop.sh
```

## Troubleshooting

### Compose still tries to pull an OctoLink service image

Confirm you are using `deploy/compose/docker-compose.yaml` from this fork and that the target
service contains both:

- `build:` pointing at the local source directory
- `pull_policy: never`

### NATS TLS startup errors

Re-generate the local test certificates and restart the stack:

```bash
rm -f nats_config/*.pem nats_config/*.key
./gen-test-certs.sh
```

### macOS host cannot validate Linux container builds

The Go services are expected to build inside Linux containers. On macOS, use Docker image builds
for final validation instead of relying on host-native `go test ./...` for service compatibility.
