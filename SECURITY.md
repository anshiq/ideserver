# Security

## Threat model (v1, single host)

- **Tenants are untrusted.** Instances can mine, scan, spam. Controls: approval
  signup (`SIGNUP_MODE`, default `approval`), per-user quotas, CPU/bandwidth
  caps (`limits.cpu/memory`, NIC ingress/egress), **port 25 blocked**, egress
  ACL (deny RFC1918, 169.254/16, host internal ports), audit log, suspend.
- **Instances are LXC containers, not KVM VMs** (most hosts this runs on are
  themselves rented cloud VMs with no `/dev/kvm` for nested virtualization —
  see `docs/adr/0005-containers-over-vms.md`). They share the host kernel, so
  the isolation boundary is namespaces/cgroups, not a hypervisor. Incus runs
  them **unprivileged** by default (root inside the container maps to an
  unprivileged UID/GID range on the host, via `idmap`) with AppArmor
  confinement and seccomp filtering — a kernel exploit is still the residual
  risk a hypervisor boundary would reduce, so container-breakout CVEs get
  patched same-day and `security.privileged` must never be set on a tenant
  profile.
- **Container escape → host.** Only `hostd` has the Incus socket (`incus-admin`
  group). `api`/`edge`/`sshgw` run unprivileged (`CAP_NET_BIND_SERVICE` only
  for the public listeners). Internal services bind loopback only.
- **Lateral movement.** NIC `port_isolation`, `mac/ipv4_filtering`; per-NIC
  ACLs on mapped ports; edge/sshgw enforce firewall where the client IP is
  visible.
- **Cookie tossing on the shared apex.** Dashboard on `app.{domain}` with
  host-only `__Host-session` (Secure, HttpOnly, SameSite=Lax), never apex-
  scoped; VM content on `*.vps.{domain}`. CSRF: Origin check + required
  `X-Requested-With` on mutations. `vps.{domain}` PSL submission tracked.
- **SSH.** Jump gateway authenticates the user's registered key, checks
  ownership + port-22 rule, then splices to the instance's private IP:22 —
  E2E encrypted with its own sshd. Host admin sshd is on **2222 key-only**;
  plain gateway logins get a help banner, never a shell.

## Secrets

- Never commit `.env`, `*.pem`, `*.key`. `ADMIN_GITHUB_LOGINS`,
  `GITHUB_CLIENT_SECRET`, `SESSION_SECRET` come from env/files only.
- Leaked legacy secrets (Gmail app password, Mongo Atlas user, old JWT
  secret) were rotated/revoked in Phase 0.

## Reporting

Report vulnerabilities privately to the maintainers; include repro steps and
impact. Do not open public issues for active exploits.
