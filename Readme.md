# VPS Platform (formerly IDEServer)

Persistent Linux machines, like AWS EC2, on **one NixOS host with one public IPv4**:
Ubuntu/Fedora **Incus LXC containers** (see `docs/adr/0005-containers-over-vms.md`
for why not KVM VMs) that survive reboots, reached over SSH and
`{id}.vps.{domain}`. The k8s code-server product is **deprecated and deleted**.

## Architecture

```
Internet → 1× IPv4 (NixOS)
 :22 → sshgw (Go jump host, ProxyJump + per-VPS fallback port)
 :80/:443 (+3000,5000,8000,8080) → edge (Go: Host/SNI routing, wildcard TLS)
 app.{domain} → web (Next.js :13000) + /api → api (Go :8081, huma+chi)
 api → Postgres 17 (+River jobs) → hostd (Go: Incus socket, reconciler, console)
 Incus bridge vpsbr0 10.100.0.0/16, ZFS pool, per-NIC ACLs + port isolation
```

| Binary | Listens | Does | Privilege |
|---|---|---|---|
| `cmd/edge` | `:80`,`:443`, extra HTTP | SNI peek, Host routing, wildcard TLS (certmagic DNS-01), firewall for hostname ports | `CAP_NET_BIND_SERVICE` |
| `cmd/sshgw` | `:22` | Jump host (x/crypto/ssh), ownership + port-22 firewall, E2E to VM sshd | `CAP_NET_BIND_SERVICE` |
| `cmd/api` | `127.0.0.1:8081` | REST `/api/v1`, GitHub OAuth, SSE, WS console relay. Never touches Incus | none |
| `cmd/hostd` | `/run/vps/hostd.sock` | River workers, reconciler (Incus events + 60s diff), console bridge, capacity | `incus-admin` |

Key flows (§1.5): `POST /instances` = one Postgres tx (quota lock, IP + SSH
fallback port alloc, `pending` row, River job in-tx). Workers converge
idempotently → `running`. `NOTIFY instance_events` → SSE + route refresh.

## Layout

```
cmd/{api,hostd,edge,sshgw}/main.go
internal/{config,db,auth,api,instances,jobs,incusx,cloudinit,netalloc,routes,edge,sshgw,ids}
web/                  Next.js dashboard (from src/frontend, rebuilt for instances)
deploy/nixos/module.nix   services.vps.* (incus preseed, bridge, zfs, postgres, nftables, units)
deploy/compose.dev.yaml   postgres only
docs/adr/             0001-incus-over-k8s … 0005-containers-over-vms
scripts/e2e.sh        end-to-end checklist (plan §6)
```

## Quickstart (dev)

```bash
just deps-up                 # postgres:17 on :5432 (vps/vps)
export DATABASE_URL="postgres://vps:vps@localhost:5432/vps?sslmode=disable"
export DOMAIN=example.com GITHUB_CLIENT_ID=… GITHUB_CLIENT_SECRET=… SESSION_SECRET=…
just migrate                 # goose up (seeds plans + images)
just dev api                 # or: go run ./cmd/api | hostd | edge | sshgw
just test                    # go test -race ./...
```

Host install: import `nixosModules.default`, set `services.vps.*` (see
`deploy/nixos/module.nix`), move admin sshd to **2222 key-only first**, then
deploy. Wildcard DNS: one A record `*.vps.{domain}` → host IP.

## SSH

```
Host *.vps.example.com
    ProxyJump vps.example.com
    User ubuntu   # or fedora
ssh a7k2m9xq.vps.example.com
ssh -p 30142 ubuntu@a7k2m9xq.vps.example.com   # fallback
```

Host keys are generated per instance (cloud-init) — dashboard shows fingerprints, no TOFU.

## Docs

- `docs/adr/` — the five pivot decisions.
- `SECURITY.md` — threat model, abuse controls, cookie rules.
- `scripts/e2e.sh` — §6 verification checklist.
