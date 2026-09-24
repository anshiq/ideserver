-- name: CreateInstance :one
INSERT INTO instances (id, user_id, name, plan_id, image_id, vcpu, memory_mb,
  disk_gb, state, desired_state, private_ipv4, https_mode, user_data)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) RETURNING *;

-- name: GetInstance :one
SELECT * FROM instances WHERE id = $1 AND deleted_at IS NULL;

-- name: GetInstanceOwned :one
SELECT * FROM instances WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL;

-- name: ListInstances :many
SELECT * FROM instances WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC;

-- name: ListInstancesByState :many
SELECT * FROM instances WHERE state = $1 AND deleted_at IS NULL;

-- name: SetInstanceState :exec
UPDATE instances SET state = $2, last_error = $3, updated_at = now() WHERE id = $1;

-- name: RenameInstance :exec
UPDATE instances SET name = $2, updated_at = now() WHERE id = $1;

-- name: SetHTTPSMode :exec
UPDATE instances SET https_mode = $2, updated_at = now() WHERE id = $1;

-- name: SetHostKeyFPs :exec
UPDATE instances SET host_key_fps = $2, updated_at = now() WHERE id = $1;

-- name: ApplyResize :exec
UPDATE instances SET vcpu=$2, memory_mb=$3, disk_gb=$4, plan_id=$5, updated_at = now()
WHERE id = $1;

-- name: SoftDeleteInstance :exec
UPDATE instances SET state='deleted', deleted_at = now() WHERE id = $1;

-- name: AttachSSHKey :exec
INSERT INTO instance_ssh_keys (instance_id, ssh_key_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: ReplaceFirewall :exec
WITH del AS (DELETE FROM firewall_rules WHERE instance_id = $1)
SELECT 1;

-- name: AddFirewallRule :exec
INSERT INTO firewall_rules (instance_id, proto, port_from, port_to, source_cidr)
VALUES ($1,$2,$3,$4,$5);

-- name: ListFirewall :many
SELECT * FROM firewall_rules WHERE instance_id = $1 ORDER BY id;

-- name: AddPortMapping :one
INSERT INTO port_mappings (vps_id, public_port, target_port, proto, kind)
VALUES ($1,$2,$3,$4,$5) RETURNING *;

-- name: ListPortMappings :many
SELECT * FROM port_mappings WHERE vps_id = $1 ORDER BY public_port;

-- name: DeletePortMapping :execrows
DELETE FROM port_mappings WHERE id = $1 AND vps_id = $2;

-- name: GetPortMappingByPublic :one
SELECT * FROM port_mappings WHERE public_port = $1;

-- name: UsedPublicPorts :many
SELECT public_port FROM port_mappings;

-- name: UsedPrivateIPs :many
SELECT private_ipv4 FROM instances WHERE deleted_at IS NULL;

-- name: Audit :exec
INSERT INTO audit_log (actor_id, instance_id, action, meta, ip)
VALUES ($1,$2,$3,$4,$5);

-- name: RecordUsage :exec
INSERT INTO usage_records (instance_id, vcpu, memory_mb, disk_gb, hour)
VALUES ($1,$2,$3,$4,$5)
ON CONFLICT (instance_id, hour) DO NOTHING;

-- name: UpsertCapacity :exec
INSERT INTO host_capacity (id, total_vcpu, free_vcpu, total_memory_mb,
  free_memory_mb, total_disk_gb, free_disk_gb, reported_at)
VALUES (1,$1,$2,$3,$4,$5,$6,now())
ON CONFLICT (id) DO UPDATE SET total_vcpu=$1, free_vcpu=$2,
  total_memory_mb=$3, free_memory_mb=$4, total_disk_gb=$5,
  free_disk_gb=$6, reported_at=now();

-- name: GetCapacity :one
SELECT * FROM host_capacity WHERE id = 1;

-- name: ListPlans :many
SELECT * FROM plans WHERE active ORDER BY vcpu, memory_mb;

-- name: GetPlan :one
SELECT * FROM plans WHERE slug = $1 AND active;

-- name: ListImages :many
SELECT * FROM images WHERE active ORDER BY distro, version;

-- name: GetImage :one
SELECT * FROM images WHERE slug = $1 AND active;

-- name: RoutesSnapshot :many
SELECT i.id, i.private_ipv4::text AS ip, i.state, i.https_mode, i.user_id
FROM instances i WHERE i.deleted_at IS NULL;
