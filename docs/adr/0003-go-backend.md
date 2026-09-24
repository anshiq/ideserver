# Status: accepted (vps-pivot)

## Context

The old backend was Node/Express + Go/gRPC runner with MongoDB, JWTs in
localStorage, and email/password auth.

## Decision

**Go for every backend service** (one module, four binaries: api, hostd,
edge, sshgw). Key libraries: Incus client, pgx, sqlc, goose, River, huma+chi,
x/oauth2, x/crypto/ssh, certmagic+libdns, coder/websocket, caarlos0/env,
x/time/rate, prometheus/client_golang. Frontend stays Next.js.

## Consequences

- One toolchain, static binaries, systemd units from the NixOS module.
- Typed OpenAPI 3.1 from huma → typed TS client for web.
- `hostd` owns Incus; `api` only writes desired state + enqueues jobs.
