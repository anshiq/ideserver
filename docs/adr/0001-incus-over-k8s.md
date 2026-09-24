# Status: accepted (vps-pivot); instance type amended by ADR-0005

## Context

The k8s product (code-server pods, OpenResty wildcard proxy, idle killer)
is deprecated. The new product is persistent Linux machines on one host.

## Decision

**Incus** manages the instances on the single NixOS host (bridge `vpsbr0`,
ZFS pool), instead of Kubernetes pods. This ADR originally specified real
KVM virtual machines; ADR-0005 changes that to LXC system containers (the
Incus/ZFS/bridge/single-host shape below is unaffected either way).

## Consequences

- Users get systemd, arbitrary ports, and (per ADR-0005) a container with
  Docker-in-Docker caveats rather than a full VM's unrestricted kernel access.
- One host is a single point of failure; mitigated with mirrored ZFS,
  off-host `pg_dump`, and a "no backups in v1" notice. Snapshots (ZFS send
  to S3) come after v1.
- `hostd` is the only process with the Incus socket; api never touches Incus.
- Multi-node later = Incus cluster + one hostd per node + River queues per node.
