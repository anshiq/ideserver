# Status: accepted (vps-pivot, amends ADR-0001)

## Context

ADR-0001 specified real KVM virtual machines. That assumed the platform runs
on a bare-metal or dedicated host with `/dev/kvm` available. The actual
target is a **rented cloud VM** (DigitalOcean/AWS/GCP/etc.), and standard
cloud VMs do not expose `/dev/kvm` to the guest — nested virtualization is
disabled by default on nearly every provider's regular instance types.
Without it, Incus's `virtual-machine` instance type cannot boot at all.

## Decision

Instances are **Incus LXC system containers** instead of KVM VMs
(`internal/incusx/real.go`'s `EnsureInstance`, one line:
`api.InstanceTypeContainer` instead of `api.InstanceTypeVM`). Nothing else
changes: the same image aliases (`images:ubuntu/24.04/cloud`, etc.), the
same `cloud-init.vendor-data`/`user-data` config keys, the same NIC/disk
device schema, and the same ZFS storage pool all work identically for
containers — Incus's API abstracts the instance type behind one field, not
two code paths. `hostd`, the DB schema, the API, and the dashboard are
unaffected.

Containers run **unprivileged** (Incus's default): root inside maps to an
unprivileged UID/GID range on the host, confined by AppArmor and seccomp.
`security.privileged` must never be set on the `default`/`vps` profile.

## Consequences

- Works on any Linux kernel with cgroups/namespaces — i.e. any cloud VM,
  regardless of provider or nested-virt support. This was the actual blocker
  for ADR-0001's plan and is why this ADR exists.
- Shared host kernel: no custom kernel or out-of-tree kernel modules inside
  an instance; Docker-in-Docker works but needs care (nesting flags, cgroup
  v2 delegation) that a real VM wouldn't.
- Isolation is namespaces/cgroups/AppArmor/seccomp, not a hypervisor
  boundary — SECURITY.md's threat model documents this explicitly. A kernel
  container-escape CVE is the residual risk; patch same-day.
- If a future host *does* have `/dev/kvm` (e.g. a dedicated server), nothing
  here prevents adding VM support back as a per-plan or per-image option —
  it is a one-line `Type` change plus a host-capability check, not a rewrite.
