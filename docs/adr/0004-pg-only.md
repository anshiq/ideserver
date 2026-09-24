# Status: accepted (vps-pivot)

## Context

MongoDB/Mongoose held users/services; Redis held cache/jobs; MinIO and
Mailpit covered artifacts and mail.

## Decision

**PostgreSQL 17 only** (goose migrations, sqlc over pgx). River job tables
live in the same DB, giving transactional enqueue (create instance + enqueue
provision in one tx — fixes the old R-04 orphan bug). LISTEN/NOTIFY replaces
Redis pub/sub for `instance_events` fan-out.

**GitHub OAuth only** (x/oauth2 + PKCE, opaque `__Host-session` cookies,
Origin + custom-header CSRF). Email/password, nodemailer, JWTs removed.

## Consequences

- One stateful service to back up (`pg_dump` nightly, off-host).
- Quotas in the DB (approval signup default for anti-abuse); usage metered
  hourly into `usage_records` for billing later.
- No CORS: `/api/*` proxies same-origin from `app.{domain}`.
