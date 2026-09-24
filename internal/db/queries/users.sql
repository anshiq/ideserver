-- name: GetUserByGithubID :one
SELECT * FROM users WHERE github_id = $1;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (github_id, login, name, email, avatar_url, role, status)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING *;

-- name: UpdateUserStatus :exec
UPDATE users SET status = $2 WHERE id = $1;

-- name: UpdateUserRole :exec
UPDATE users SET role = $2 WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY id LIMIT $1 OFFSET $2;

-- name: EnsureQuota :one
INSERT INTO quotas (user_id) VALUES ($1)
ON CONFLICT (user_id) DO NOTHING RETURNING *;

-- name: GetQuota :one
SELECT * FROM quotas WHERE user_id = $1;

-- name: UpdateQuota :exec
UPDATE quotas SET max_instances=$2, max_vcpu=$3, max_memory_mb=$4,
  max_disk_gb=$5, max_mapped_ports=$6 WHERE user_id = $1;

-- name: UserUsage :one
SELECT
  COUNT(i.id)::int AS instances,
  COALESCE(SUM(i.vcpu),0)::int AS vcpu,
  COALESCE(SUM(i.memory_mb),0)::int AS memory_mb,
  COALESCE(SUM(i.disk_gb),0)::int AS disk_gb,
  (SELECT COUNT(*)::int FROM port_mappings p JOIN instances i2
    ON i2.id = p.vps_id WHERE i2.user_id = $1 AND i2.deleted_at IS NULL) AS mapped_ports
FROM instances i WHERE i.user_id = $1 AND i.deleted_at IS NULL;
