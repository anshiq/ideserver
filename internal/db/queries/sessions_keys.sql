-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at, ip, user_agent)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetSession :one
SELECT s.*, u.id AS u_id, u.github_id, u.login, u.name, u.email,
  u.avatar_url, u.role, u.status, u.created_at AS u_created
FROM sessions s JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token_hash = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;

-- name: UpsertSSHKey :one
INSERT INTO ssh_keys (user_id, name, public_key, fingerprint)
VALUES ($1, $2, $3, $4)
ON CONFLICT (fingerprint) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: ListSSHKeys :many
SELECT * FROM ssh_keys WHERE user_id = $1 ORDER BY id;

-- name: DeleteSSHKey :execrows
DELETE FROM ssh_keys WHERE id = $1 AND user_id = $2;

-- name: InstanceSSHKeys :many
SELECT k.* FROM ssh_keys k
JOIN instance_ssh_keys ik ON ik.ssh_key_id = k.id
WHERE ik.instance_id = $1;
