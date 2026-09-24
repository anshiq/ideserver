# Status: accepted (vps-pivot)

## Context

One server, one public IPv4. HTTP can route by Host header and TLS by SNI,
but SSH carries no hostname.

## Decision

- **HTTP/TLS** route by Host/SNI at the `edge` (`{id}.vps.{domain}` → VM,
  `app.{domain}` → dashboard). Wildcard TLS via certmagic DNS-01 (libdns).
- **SSH** goes through a **jump gateway** (`sshgw` on :22, ProxyJump), plus
  a per-VPS fallback public port (30000-39999) for zero-config access.
- **Raw TCP/UDP** go through mapped public ports (Incus network forwards /
  DNAT, client IP preserved).
- Host admin sshd moves to **2222 key-only** before 22 is freed.

## Consequences

- One wildcard A record (`*.vps.{domain}`) — VPSes never need DNS changes.
- `vps.{domain}` (gateway) must never match `*.vps.{domain}` (recursion guard
  in `SplitHost`).
- Firewall enforced at three points: edge (hostname ports), sshgw (port 22),
  NIC ACL (mapped ports).
