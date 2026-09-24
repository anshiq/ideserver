-- +goose Up
-- VPS platform baseline schema (PostgreSQL 17).

CREATE TABLE users (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  github_id   BIGINT NOT NULL UNIQUE,
  login       TEXT NOT NULL,
  name        TEXT NOT NULL DEFAULT '',
  email       TEXT NOT NULL DEFAULT '',
  avatar_url  TEXT NOT NULL DEFAULT '',
  role        TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('user','admin')),
  status      TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','active','suspended')),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash  BYTEA NOT NULL UNIQUE,
  expires_at  TIMESTAMPTZ NOT NULL,
  ip          TEXT NOT NULL DEFAULT '',
  user_agent  TEXT NOT NULL DEFAULT '',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sessions_user_idx ON sessions(user_id);

CREATE TABLE ssh_keys (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name        TEXT NOT NULL,
  public_key  TEXT NOT NULL,
  fingerprint TEXT NOT NULL UNIQUE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX ssh_keys_user_idx ON ssh_keys(user_id);

CREATE TABLE plans (
  slug      TEXT PRIMARY KEY,
  vcpu      INT NOT NULL,
  memory_mb INT NOT NULL,
  disk_gb   INT NOT NULL,
  active    BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE images (
  slug         TEXT PRIMARY KEY,
  distro       TEXT NOT NULL,
  version      TEXT NOT NULL,
  incus_alias  TEXT NOT NULL,
  default_user TEXT NOT NULL,
  active       BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE quotas (
  user_id          BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  max_instances    INT NOT NULL DEFAULT 2,
  max_vcpu         INT NOT NULL DEFAULT 2,
  max_memory_mb    INT NOT NULL DEFAULT 4096,
  max_disk_gb      INT NOT NULL DEFAULT 40,
  max_mapped_ports INT NOT NULL DEFAULT 5
);

CREATE TABLE instances (
  id            TEXT PRIMARY KEY, -- 8-char VPS id
  user_id       BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name          TEXT NOT NULL,
  plan_id       TEXT NOT NULL REFERENCES plans(slug),
  image_id      TEXT NOT NULL REFERENCES images(slug),
  vcpu          INT NOT NULL,
  memory_mb     INT NOT NULL,
  disk_gb       INT NOT NULL,
  state         TEXT NOT NULL DEFAULT 'pending',
  desired_state TEXT NOT NULL DEFAULT 'running',
  private_ipv4  INET NOT NULL UNIQUE,
  https_mode    TEXT NOT NULL DEFAULT 'terminate' CHECK (https_mode IN ('terminate','passthrough')),
  host_key_fps  TEXT NOT NULL DEFAULT '',
  user_data     TEXT NOT NULL DEFAULT '',
  last_error    TEXT NOT NULL DEFAULT '',
  deleted_at    TIMESTAMPTZ,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX instances_user_idx ON instances(user_id) WHERE deleted_at IS NULL;
CREATE INDEX instances_state_idx ON instances(state) WHERE deleted_at IS NULL;

CREATE TABLE instance_ssh_keys (
  instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
  ssh_key_id  BIGINT NOT NULL REFERENCES ssh_keys(id) ON DELETE RESTRICT,
  PRIMARY KEY (instance_id, ssh_key_id)
);

CREATE TABLE firewall_rules (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
  proto       TEXT NOT NULL CHECK (proto IN ('tcp','udp')),
  port_from   INT NOT NULL,
  port_to     INT NOT NULL,
  source_cidr CIDR NOT NULL,
  CHECK (port_from BETWEEN 1 AND 65535 AND port_to BETWEEN port_from AND 65535)
);
CREATE INDEX firewall_instance_idx ON firewall_rules(instance_id);

CREATE TABLE port_mappings (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  vps_id      TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
  public_port INT NOT NULL UNIQUE CHECK (public_port BETWEEN 30000 AND 39999),
  target_port INT NOT NULL CHECK (target_port BETWEEN 1 AND 65535),
  proto       TEXT NOT NULL CHECK (proto IN ('tcp','udp')),
  kind        TEXT NOT NULL DEFAULT 'custom' CHECK (kind IN ('ssh-fallback','custom')),
  UNIQUE (vps_id, target_port, proto)
);
CREATE INDEX portmap_vps_idx ON port_mappings(vps_id);

CREATE TABLE audit_log (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  actor_id    BIGINT REFERENCES users(id) ON DELETE SET NULL,
  instance_id TEXT REFERENCES instances(id) ON DELETE SET NULL,
  action      TEXT NOT NULL,
  meta        JSONB NOT NULL DEFAULT '{}',
  ip          TEXT NOT NULL DEFAULT '',
  at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_instance_idx ON audit_log(instance_id);

CREATE TABLE usage_records (
  id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  instance_id TEXT NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
  vcpu        INT NOT NULL,
  memory_mb   INT NOT NULL,
  disk_gb     INT NOT NULL,
  hour        TIMESTAMPTZ NOT NULL,
  UNIQUE (instance_id, hour)
);

CREATE TABLE host_capacity (
  id             INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  total_vcpu     INT NOT NULL,
  free_vcpu      INT NOT NULL,
  total_memory_mb INT NOT NULL,
  free_memory_mb INT NOT NULL,
  total_disk_gb  INT NOT NULL,
  free_disk_gb   INT NOT NULL,
  reported_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Fan-out channel: every state change wakes api (SSE) + edge/sshgw (routes).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION notify_instance_event() RETURNS trigger AS $$
BEGIN
  PERFORM pg_notify('instance_events', NEW.id || ':' || NEW.state);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER instances_notify AFTER INSERT OR UPDATE OF state ON instances
  FOR EACH ROW EXECUTE FUNCTION notify_instance_event();

-- Seed catalog.
INSERT INTO plans (slug, vcpu, memory_mb, disk_gb) VALUES
  ('xs', 1, 1024, 10),
  ('s',  2, 2048, 20),
  ('m',  2, 4096, 40),
  ('l',  4, 8192, 80);

INSERT INTO images (slug, distro, version, incus_alias, default_user) VALUES
  ('ubuntu-24.04', 'ubuntu', '24.04', 'images:ubuntu/24.04/cloud', 'ubuntu'),
  ('ubuntu-22.04', 'ubuntu', '22.04', 'images:ubuntu/22.04/cloud', 'ubuntu'),
  ('fedora-42',    'fedora', '42',    'images:fedora/42/cloud',    'fedora'),
  ('fedora-41',    'fedora', '41',    'images:fedora/41/cloud',    'fedora');

-- +goose Down
DROP TRIGGER IF EXISTS instances_notify ON instances;
DROP FUNCTION IF EXISTS notify_instance_event();
DROP TABLE IF EXISTS usage_records, audit_log, port_mappings, firewall_rules,
  instance_ssh_keys, instances, quotas, images, plans, ssh_keys, sessions, users,
  host_capacity;
