# WS-22 Server Build + Smoke Test Record

**Date:** 2026-06-21  
**Server:** `39.97.250.156` (2 cores / 1.6 GB RAM / no swap / 22 GB disk)  
**Branch:** `agent/cloud-guru/e4accf20` (PR #13)  
**Final commit:** `94f0e22`

---

## Build Summary

All 12 custom services built from local source using `docker compose build` (sequential, one at a time due to server resource constraints):

| Service | Image | Size |
|---------|-------|------|
| controller | octolink/controller:local | 40.4 MB |
| acs | octolink/acs:local | 17.4 MB |
| adapter | octolink/adapter:local | 17.8 MB |
| ws | octolink/ws:local | 21.2 MB |
| ws-adapter | octolink/ws-adapter:local | 22.6 MB |
| mqtt | octolink/mqtt:local | 19.8 MB |
| mqtt-adapter | octolink/mqtt-adapter:local | 20.1 MB |
| stomp | octolink/stomp:local | 18.3 MB |
| stomp-adapter | octolink/stomp-adapter:local | 20.0 MB |
| file-server | octolink/file-server:local | 39.2 MB |
| socketio | octolink/socketio:local | 207 MB |
| frontend | octolink/frontend:local | 1.66 GB |

`pull_policy: never` confirmed — no fallback to upstream `oktopusp/*` images.

---

## Issues Encountered and Fixed

### 1. Go version mismatch
- **Error:** `go.mod requires go >= 1.23.0 (running go 1.22.12)`
- **Fix:** All Go Dockerfiles updated from `golang:1.22` → `golang:1.24` (commit `fff4a3b`)

### 2. Go proxy blocked (China server)
- **Error:** `proxy.golang.org` TLS timeout
- **Fix:** Added `ENV GOPROXY=https://goproxy.cn,direct` to all Go Dockerfiles (commit `3af05f5`)

### 3. NATS cert missing SAN
- **Error:** `bad certificate` — cert had `CN=nats` but clients connect to `msg_broker:4222`
- **Fix:** Regenerated cert with SANs: `DNS:msg_broker, DNS:nats, DNS:localhost, IP:127.0.0.1`
- **Committed:** `gen-test-certs.sh` updated with `-extfile` SAN extension (this PR)

### 4. NATS_ENABLE_TLS quoted string
- **Error:** `strconv.ParseBool: parsing "\"true\"": invalid syntax`
- **Fix:** All `.env.*.example` files changed `NATS_ENABLE_TLS="true"` → `NATS_ENABLE_TLS=true` (commit `a7690f2`)

### 5. bson.M compound index
- **Error:** `multi-key map passed in for ordered parameter keys` (db.go:78)
- **Fix:** Changed `bson.M{"name": 1, "tenant_id": 1}` → `bson.D{{Key: "name", Value: 1}, {Key: "tenant_id", Value: 1}}` (commit `94f0e22`)

---

## Stack Startup

```
docker compose ps (after all fixes)
NAME          IMAGE                        STATUS         PORTS
controller    octolink/controller:local    Up ~1min       0.0.0.0:8000->8000/tcp
file-server   octolink/file-server:local   Up 12min       0.0.0.0:8004->8004/tcp
frontend      octolink/frontend:local      Up 12min       0.0.0.0:3000->3000/tcp
mongo_usp     mongo                        Up 12min       0.0.0.0:27017->27017/tcp
nats          nats:latest                  Up 11min       0.0.0.0:4222->4222/tcp, 0.0.0.0:8222->8222/tcp
nginx         nginx:latest                 Up 12min       0.0.0.0:80->80/tcp
socketio      octolink/socketio:local      Up 11min       0.0.0.0:5000->5000/tcp
```

---

## Smoke Test Results

### S3 — Frontend OctoLink Branding ✅

```
curl http://localhost:3000/_next/static/chunks/pages/_app-5903d47ccfa1a4a4.js
→ grep: "OctoLink" (1 match, no "OktopUSP")
```

Frontend JS bundle contains `OctoLink` branding — upstream `OktopUSP` replaced.

### S7 — RBAC: Unauthenticated 401 ✅

```
curl http://localhost:8000/api/users
→ HTTP 401
```

### S7 — Login returns JWT with `tenant_id` + `role` ✅

```
curl -X PUT http://localhost:8000/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"Admin123!"}'

JWT payload:
{
  "username": "",
  "email": "",
  "role": "super_admin",
  "tenant_id": "default",
  "iss": "OctoLink",
  "exp": 1782145056
}
```

`iss: "OctoLink"` confirms we are running OctoLink's custom JWT issuer (not upstream `OktopUSP`).

### S7 — Authenticated user list ✅

```
curl -H "Authorization: <token>" http://localhost:8000/api/users
→ [{"role":"super_admin","tenant_id":"default",...}]
HTTP 200
```

---

## Changelog

- `ec95f64` — feat(build): restore image build tooling + compose local-source build + test cert script
- `fff4a3b` — fix(build): upgrade Go builder to 1.24
- `3af05f5` — fix(build): add GOPROXY=goproxy.cn + npm CN mirror for China server network
- `a7690f2` — fix(deploy): remove quotes from NATS_ENABLE_TLS in all .env examples
- `94f0e22` — fix(controller): use bson.D for compound index
- *(this commit)* — fix(deploy): add SAN extensions to gen-test-certs.sh + smoke test docs
